package services

import (
	ctx "context"
	"errors"
	"fmt"
	"log"
	"os"
	"strings"

	nft_proxy "github.com/alphabatem/nft-proxy"
	"github.com/alphabatem/nft-proxy/metaplex_core"
	token_metadata "github.com/alphabatem/nft-proxy/token-metadata"
	"github.com/alphabatem/token_2022_go"
	"github.com/babilu-online/common/context"
	bin "github.com/gagliardetto/binary"
	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/rpc"
)

// Added context (ctx.Context) to all RPC calls, allowing better control of timeouts and cancellations.

type SolanaService struct {
	context.DefaultService
	client *rpc.Client
}

const SOLANA_SVC = "solana_svc"

func (svc SolanaService) Id() string {
	return SOLANA_SVC
}

// Start initializes the RPC client for Solana with error handling for invalid RPC URLs
func (svc *SolanaService) Start() error {
	rpcURL := os.Getenv("RPC_URL")
	if rpcURL == "" {
		return fmt.Errorf("RPC_URL is not configured")
	}

	svc.client = rpc.New(rpcURL)
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

func (svc *SolanaService) TokenData(key solana.PublicKey) (*token_metadata.Metadata, uint8, error) {
	var meta token_metadata.Metadata
	var mint token_2022.Mint

	ata, _, _ := svc.FindTokenMetadataAddress(key, solana.TokenMetadataProgramID)
	ataT22, _, _ := svc.FindTokenMetadataAddress(key, solana.MustPublicKeyFromBase58("META4s4fSmpkTbZoUsgC1oBnWB31vQcmnN8giPw51Zu"))

	accs, err := svc.client.GetMultipleAccountsWithOpts(ctx.TODO(), []solana.PublicKey{key, ata, ataT22}, &rpc.GetMultipleAccountsOpts{Commitment: rpc.CommitmentProcessed})
	if err != nil {
		return nil, 0, err
	}

	var decimals uint8
	if accs.Value[0] != nil {
		//log.Printf("SolanaService::TokenData:%s - Owner: %s", key, accs.Value[0].Owner)

		err := mint.UnmarshalWithDecoder(bin.NewBinDecoder(accs.Value[0].Data.GetBinary()))
		if err == nil {
			decimals = mint.Decimals
		}

		switch accs.Value[0].Owner {
		case nft_proxy.METAPLEX_CORE:
			_meta, err := svc.decodeMetaplexCoreMetadata(key, accs.Value[0].Data.GetBinary())
			if err != nil {
				return nil, decimals, err
			}

			if _meta != nil {
				return _meta, decimals, nil
			}
		case nft_proxy.TOKEN_2022:
			exts, err := mint.Extensions()
			if err != nil {
				log.Printf("T22 Ext err: %s", err)
				break
			}
			if exts != nil && exts.TokenMetadata != nil {
				return &token_metadata.Metadata{
					// Put right name convention
					Protocol:        token_metadata.ProtocolToken22Mint,
					UpdateAuthority: *exts.TokenMetadata.Authority,
					Mint:            exts.TokenMetadata.Mint,
					Data: token_metadata.Data{
						Name:   exts.TokenMetadata.Name,
						Symbol: exts.TokenMetadata.Symbol,
						Uri:    exts.TokenMetadata.Uri,
					},
				}, decimals, nil
			}
		}
	}

	for _, acc := range accs.Value[1:] {
		if acc == nil {
			continue
		}

		err := bin.NewBorshDecoder(acc.Data.GetBinary()).Decode(&meta)
		if err != nil {
			log.Printf("Decode err: %s", err)
			continue
		}
		return &meta, decimals, nil
	}

	return nil, decimals, errors.New("unable to find token metadata")
}

func (svc *SolanaService) decodeMintMetadata(data []byte) (*token_metadata.Metadata, error) {
	var mint token_2022_go.Mint
	err := mint.UnmarshalWithDecoder(bin.NewBinDecoder(data))
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
				Protocol:        token_metadata.ProtocolToken22Mint,
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

	return nil, nil
}

func (svc *SolanaService) decodeMetaplexCoreMetadata(mint solana.PublicKey, data []byte) (*token_metadata.Metadata, error) {
	var meta metaplex_core.Asset
	err := meta.UnmarshalWithDecoder(bin.NewBinDecoder(data))
	if err != nil {
		return nil, err
	}

	log.Printf("%+v\n", meta)

	tMeta := token_metadata.Metadata{
		Protocol: token_metadata.ProtocolMetaplexCore,
		Mint:     mint,
		Data: token_metadata.Data{
			Name: strings.Trim(meta.Name, "\x00"),
			Uri:  strings.Trim(meta.Uri, "\x00"),
		},
	}

	if meta.UpdateAuthority != nil {
		tMeta.UpdateAuthority = *meta.UpdateAuthority
	}

	return &tMeta, nil
}

func (svc *SolanaService) CreatorKeys(tokenMint solana.PublicKey) ([]solana.PublicKey, error) {
	metadata, _, err := svc.TokenData(tokenMint)
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
