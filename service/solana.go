package services

import (
	ctx "context"
	"errors"
	"github.com/alphabatem/nft-proxy/token_2022"
	"github.com/babilu-online/common/context"
	bin "github.com/gagliardetto/binary"
	token_metadata "github.com/gagliardetto/metaplex-go/clients/token-metadata"
	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/rpc"
	"log"
	"os"
)

type SolanaService struct {
	context.DefaultService
	client *rpc.Client
}

const SOLANA_SVC = "solana_svc"

func (svc SolanaService) Id() string {
	return SOLANA_SVC
}

func (svc *SolanaService) Start() error {
	svc.client = rpc.New(os.Getenv("RPC_URL"))

	return nil
}

func (svc *SolanaService) Client() *rpc.Client {
	return svc.client
}

func (svc *SolanaService) RecentBlockhash() (solana.Hash, error) {
	bhash, err := svc.Client().GetRecentBlockhash(ctx.Background(), rpc.CommitmentFinalized)
	if err != nil {
		return solana.Hash{}, err
	}

	return bhash.Value.Blockhash, nil
}

func (svc *SolanaService) TokenData(key solana.PublicKey) (*token_metadata.Metadata, error) {
	var meta token_metadata.Metadata

	ata, _, _ := svc.FindTokenMetadataAddress(key, solana.TokenMetadataProgramID)
	ataT22, _, _ := svc.FindTokenMetadataAddress(key, solana.MustPublicKeyFromBase58("META4s4fSmpkTbZoUsgC1oBnWB31vQcmnN8giPw51Zu"))

	accs, err := svc.client.GetMultipleAccountsWithOpts(ctx.TODO(), []solana.PublicKey{key, ata, ataT22}, &rpc.GetMultipleAccountsOpts{Commitment: rpc.CommitmentProcessed})
	if err != nil {
		return nil, err
	}

	if accs.Value[0] != nil {
		var mint token_2022.Mint2022
		err = mint.UnmarshalWithDecoder(bin.NewBinDecoder(accs.Value[0].Data.GetBinary()))
		if err != nil {
			return nil, err
		}

		exts, err := mint.Extensions()
		if err != nil {
			return nil, err
		}

		if exts != nil {
			if exts.MetadataPointer != nil {
				//TODO
			}

			if exts.TokenMetadata != nil {
				return &token_metadata.Metadata{
					UpdateAuthority: *exts.TokenMetadata.Authority,
					Mint:            exts.TokenMetadata.Mint,
					Data: token_metadata.Data{
						Name:   exts.TokenMetadata.Name,
						Symbol: exts.TokenMetadata.Symbol,
						Uri:    exts.TokenMetadata.Uri,
					},
				}, nil
			}
		}

	}

	for _, acc := range accs.Value[1:] {
		if acc == nil {
			continue
		}

		err := bin.NewBorshDecoder(acc.Data.GetBinary()).Decode(&meta)
		if err == nil {
			return &meta, nil
		}
	}

	return nil, errors.New("unable to find token metadata")
}

func (svc *SolanaService) CreatorKeys(tokenMint solana.PublicKey) ([]solana.PublicKey, error) {
	metadata, err := svc.TokenData(tokenMint)
	if err != nil {
		log.Printf("%s creatorKeys err: %s", tokenMint, err)
		return nil, err
	}

	if metadata.Data.Creators == nil {
		return nil, errors.New("unable to find creators")
	}

	creatorKeys := make([]solana.PublicKey, len(*metadata.Data.Creators))
	for i, c := range *metadata.Data.Creators {
		creatorKeys[i] = c.Address
	}
	return creatorKeys, nil
}

// FindTokenMetadataAddress returns the token metadata program-derived address given a SPL token mint address.
func (svc *SolanaService) FindTokenMetadataAddress(mint solana.PublicKey, metadataProgam solana.PublicKey) (solana.PublicKey, uint8, error) {
	seed := [][]byte{
		[]byte("metadata"),
		metadataProgam[:],
		mint[:],
	}
	return solana.FindProgramAddress(seed, metadataProgam)
}
