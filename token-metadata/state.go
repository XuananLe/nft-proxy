package token_metadata

import (
	token_metadata "github.com/gagliardetto/metaplex-go/clients/token-metadata"
	"github.com/gagliardetto/solana-go"
)

type Protocol uint8  // More specific size since we only need few values

const (
	// Right name convention
    ProtocolLegacy Protocol = iota
    ProtocolToken22Mint
    ProtocolLibreplex
    ProtocolMetaplexCore
    maxProtocol // private constant for validation
)

type Metadata struct {
	Key             token_metadata.Key
	UpdateAuthority solana.PublicKey
	Mint            solana.PublicKey
	Data            Data

	// Immutable, once flipped, all sales of this metadata are considered secondary.
	PrimarySaleHappened bool

	// Whether or not the data struct is mutable, default is not
	IsMutable bool

	// Collection
	// Issues:
	// Using pointers for optional fields can lead to nil pointer dereferences
	// No validation for optional fields
	// No documentation about nil behavior
	
	Collection *token_metadata.Collection `bin:"optional"`


	// Issues:

	// No documentation explaining why Protocol is excluded from binary serialization
	// Missing JSON tags for API compatibility
	// No validation tags

    Protocol Protocol `bin:"-" json:"protocol"`
}


type SellerFeeBasisPoints uint16

func (s SellerFeeBasisPoints) Valid() bool {
    return 0 <= s  && s <= 10000
}

type Data struct {
	// The name of the asset
	Name string

	// The symbol for the asset
	Symbol string

	// URI pointing to JSON representing the asset
	Uri string

	// Issues:

	// No validation for SellerFeeBasisPoints range (0-10000)
	// Using pointer to slice (unnecessary indirection)
	// No field validation methods

	// Royalty basis points that goes to creators in secondary sales (0-10000)
	SellerFeeBasisPoints SellerFeeBasisPoints

	// Array of creators, optional
    Creators []token_metadata.Creator `bin:"optional" json:"creators,omitempty"`
}
