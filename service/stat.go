package services

import (
	"log"
	"sync/atomic"

	nft_proxy "github.com/alphabatem/nft-proxy"
	"github.com/babilu-online/common/context"
)

type StatService struct {
	context.DefaultService

	imageFilesServed uint64
	mediaFilesServed uint64
	requestsServed   uint64

	sql *SqliteService
}

const STAT_SVC = "stat_svc"

func (svc StatService) Id() string {
	return STAT_SVC
}

func (svc *StatService) Start() error {
	svc.sql = svc.DefaultService(SQLITE_SVC).(*SqliteService)

	return nil
}

func (svc *StatService) IncrementImageFileRequests() {
	atomic.AddUint64(&svc.imageFilesServed, 1)
}

func (svc *StatService) IncrementMediaFileRequests() {
	atomic.AddUint64(&svc.mediaFilesServed, 1)
}

func (svc *StatService) IncrementMediaRequests() {
	atomic.AddUint64(&svc.requestsServed, 1)
}

// The counters are now returned as atomically loaded values, ensuring thread-safety during stat retrieval.
func (svc *StatService) ServiceStats() (map[string]interface{}, error) {
	// Retrieve image count from the database
	var imgCount int64
	if err := svc.sql.Db().Model(&nft_proxy.SolanaMedia{}).Count(&imgCount).Error; err != nil {
		log.Printf("Error retrieving image count from database: %v", err)
		return nil, err
	}

	// Return the stats with atomically loaded values for thread safety
	return map[string]interface{}{
		"images_stored":      imgCount,
		"requests_served":    atomic.LoadUint64(&svc.requestsServed),
		"image_files_served": atomic.LoadUint64(&svc.imageFilesServed),
		"media_files_served": atomic.LoadUint64(&svc.mediaFilesServed),
	}, nil
}