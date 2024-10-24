package services

import (
	"encoding/json"
	nft_proxy "github.com/alphabatem/nft-proxy"
	token_metadata "github.com/alphabatem/nft-proxy/token-metadata"
	"github.com/babilu-online/common/context"
	"github.com/gagliardetto/solana-go"
	"gorm.io/gorm/clause"
	"io"
	"log"
	"net/http"
	"strings"
	"time"
)

type SolanaImageService struct {
	context.DefaultService
	sql *SqliteService
	sol *SolanaService

	http *http.Client
}

const SOLANA_IMG_SVC = "solana_img_svc"

func (svc SolanaImageService) Id() string {
	return SOLANA_IMG_SVC
}

func (svc *SolanaImageService) Start() error {
	svc.http = &http.Client{Timeout: 5 * time.Second}

	svc.sql = svc.DefaultService(SQLITE_SVC).(*SqliteService)
	svc.sol = svc.DefaultService(SOLANA_SVC).(*SolanaService)
	return nil
}

func (svc *SolanaImageService) Media(key string, skipCache bool) (*nft_proxy.Media, error) {
	var media *nft_proxy.SolanaMedia
	err := svc.sql.Db().First(&media, "mint = ?", key).Error
	if err != nil || skipCache {
		log.Printf("FetchMetadata - %s err: %s", key, err)
		media, err = svc.FetchMetadata(key)
		if err != nil {
			return nil, err //Still cant get metadata
		}
	}

	return media.Media(), nil
}

func (svc *SolanaImageService) RemoveMedia(key string) error {
	return svc.sql.Db().Delete(&nft_proxy.SolanaMedia{}, "mint = ?", key).Error
}

func (svc *SolanaImageService) FetchMetadata(key string) (*nft_proxy.SolanaMedia, error) {
	metadata, err := svc._retrieveMetadata(key)
	if err != nil {
		return nil, err
	}

	media, err := svc.cache(key, metadata, "")
	if err != nil {
		return nil, err
	}

	return media, nil
}

func (svc *SolanaImageService) _retrieveMetadata(key string) (*nft_proxy.NFTMetadataSimple, error) {
	pk, err := solana.PublicKeyFromBase58(key)
	if err != nil {
		return nil, err
	}
	tokenData, decimals, err := svc.sol.TokenData(pk)
	if err != nil || tokenData == nil {
		log.Printf("No token data for %s - %s", pk, err)
		return nil, err
	}

	//log.Printf("TokenData retreive (%v): %+v\n", decimals, tokenData)

	switch tokenData.Protocol {
	case token_metadata.PROTOCOL_METAPLEX_CORE:
		return &nft_proxy.NFTMetadataSimple{
			Image:           tokenData.Data.Uri,
			Decimals:        decimals,
			Name:            strings.Trim(tokenData.Data.Name, "\x00"),
			Symbol:          strings.Trim(tokenData.Data.Symbol, "\x00"),
			UpdateAuthority: tokenData.UpdateAuthority.String(),
		}, nil
	default:
		//Get file meta if possible
		f, err := svc.retrieveFile(tokenData.Data.Uri)
		if f != nil {
			f.Decimals = decimals
			f.UpdateAuthority = tokenData.UpdateAuthority.String()
			return f, nil
		}
		log.Printf("(%s) retrieveFile err: %s", tokenData.Data.Uri, err)
	}

	//No Metadata
	return &nft_proxy.NFTMetadataSimple{
		Name:            strings.Trim(tokenData.Data.Name, "\x00"),
		Decimals:        decimals,
		Symbol:          strings.Trim(tokenData.Data.Symbol, "\x00"),
		UpdateAuthority: tokenData.UpdateAuthority.String(),
	}, nil
}

func (svc *SolanaImageService) retrieveFile(uri string) (*nft_proxy.NFTMetadataSimple, error) {
	file, err := svc.http.Get(strings.Trim(uri, "\x00")) //Strip crap off urls
	if err != nil {
		return nil, err
	}

	if file.StatusCode != 200 {
		return nil, err
	}

	defer file.Body.Close()
	data, err := io.ReadAll(file.Body)
	if err != nil {
		return nil, err
	}

	var metadata nft_proxy.NFTMetadataSimple
	err = json.Unmarshal(data, &metadata)
	if err != nil {
		return nil, err
	}

	return &metadata, nil
}

func (svc *SolanaImageService) cache(key string, metadata *nft_proxy.NFTMetadataSimple, localPath string) (*nft_proxy.SolanaMedia, error) {
	media := nft_proxy.SolanaMedia{
		Mint:      key,
		LocalPath: localPath,
	}

	//log.Printf("Metadata: %+v\n", metadata)
	if metadata != nil {
		media.Name = metadata.Name
		media.Symbol = metadata.Symbol
		media.ImageUri = metadata.Image
		media.ImageType = svc.guessImageType(metadata)
		media.UpdateAuthority = metadata.UpdateAuthority
		media.MintDecimals = metadata.Decimals

		mediaFile := metadata.AnimationFile()
		if mediaFile != nil {
			media.MediaUri = mediaFile.URL
			mediaFile.Type = "mp4"
			if strings.Contains(mediaFile.Type, "/") {
				media.MediaType = strings.Split(mediaFile.Type, "/")[1]
			}
		}
	}

	return &media, svc.sql.Db().Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "mint"}}, // key colum
		UpdateAll: true,
	}).Create(&media).Error
}

func (svc *SolanaImageService) guessImageType(metadata *nft_proxy.NFTMetadataSimple) string {
	if metadata == nil {
		return "jpg"
	}

	imageType := ""
	imgFile := metadata.ImageFile()
	if imgFile != nil && strings.Contains(imgFile.Type, "/") {
		imageType = strings.Split(imgFile.Type, "/")[1]
	}
	if imageType == "" {
		parts := strings.Split(metadata.Image, ".")
		lastPart := parts[len(parts)-1]
		if strings.Contains(lastPart, "=") {
			parts := strings.Split(lastPart, "=")
			imageType = parts[len(parts)-1]
		} else {
			imageType = lastPart
		}
	}

	if strings.Contains(imageType, "?") {
		imageType = strings.Split(imageType, "?")[0]
	}

	if !svc.ValidType(imageType) {
		log.Printf("Invalid image type guessed: %s", imageType)
		return "jpg"
	}

	return imageType
}

func (svc *SolanaImageService) ValidType(imageType string) bool {
	switch imageType {
	case "png":
		return true
	case "jpg":
		return true
	case "jpeg":
		return true
	case "gif":
		return true
	case "svg":
		return true
	default:
		return false
	}
}
