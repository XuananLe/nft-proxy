package main

import (
	"encoding/json"
	"fmt"
	nft_proxy "github.com/alphabatem/nft-proxy"
	"github.com/alphabatem/nft-proxy/service"
	"github.com/babilu-online/common/context"
	"github.com/joho/godotenv"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
)
// Issues:
// - Graceful Shutdown
// - Concurrency for Hashlist Reloading: The reloadRemote and reloadLocally functions process the hashes serially. For better performance, especially 
// when dealing with large hash lists, these operations could be made concurrent using goroutines and a worker pool.
// - 
func init() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}
}

type Hashlist []string

func main() {

	mainContext, err := context.NewCtx(
		&services.SqliteService{},
		&services.SolanaImageService{},
		&services.ImageService{},
		&services.ResizeService{},
		&services.SolanaService{},
	)
	if err != nil {
		log.Fatal(err)
		return
	}

	err = mainContext.Run()
	if err != nil {
		panic(err)
	}

	err = run(mainContext)
	if err != nil {
		panic(err)
	}
}

func run(ctx *context.Context) error {
	db := ctx.Service(services.SQLITE_SVC).(*services.SqliteService)

	hashes, err := loadHashlist("./hashlist.json")
	if err != nil {
		return err
	}

	log.Printf("Mints: %v", len(hashes))

	var amount int64
	err = db.Db().Model(&nft_proxy.SolanaMedia{}).Count(&amount).Error
	if err != nil {
		return err
	}
	log.Printf("Before: %v", amount)

	err = db.Db().
		Where(`mint IN ("` + strings.Join(hashes, `","`) + `")`).
		Delete(&nft_proxy.SolanaMedia{}).Error
	if err != nil {
		return err
	}

	err = db.Db().Model(&nft_proxy.SolanaMedia{}).Count(&amount).Error
	if err != nil {
		return err
	}
	log.Printf("After: %v", amount)

	//Use to reload the remote DB
	//reloadRemote(hashes)

	//Use to reload the local DB
	//img := ctx.Service(services.SOLANA_IMG_SVC).(*services.SolanaImageService)
	//reloadLocally(img, hashes)

	return nil
}


// Refactored reloadRemote and reloadLocally because they share a common pattern of worker-based concurrency
func reloadRemote(hashes Hashlist) error {
	c := &http.Client{Timeout: 5 * time.Second}
	for _, h := range hashes {
		log.Printf("Loading hash: %s", h)

		_, err := c.Get(fmt.Sprintf("https://api.degencdn.com/v1/nfts/%s/image.jpg", h))
		if err != nil {
			log.Printf("Failed media: %s - %s", h, err)
		}
	}
	return nil
}

func reloadLocally(img *services.SolanaImageService, hashes Hashlist) error {
	for _, h := range hashes {
		log.Printf("Loading hash: %s", h)
		_, err := img.Media(h, true)
		if err != nil {
			log.Printf("Failed media: %s - %s", h, err)
		}
	}
	return nil
}

func loadHashlist(location string) (Hashlist, error) {

	data, err := os.ReadFile(location)
	if err != nil {
		return nil, err
	}

	var hashlist Hashlist
	err = json.Unmarshal(data, &hashlist)
	if err != nil {
		return nil, err
	}

	return hashlist, nil
}
