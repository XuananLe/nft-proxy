package services

import (
	"errors"
	"fmt"
	"github.com/babilu-online/common/context"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"log"
	"math/rand"
	"net/http"
	"os"
	"strconv"
	"time"
)

// @title NFT Aggregator Swap API
// @version 1.1.27
// @description NFT Aggregator Swap API
// @schemes https

// @host https://agg.alphabatem.com
// @BasePath /
// @query.collection.format multi

type HttpService struct {
	context.DefaultService
	BaseURL string
	Port    int

	imgSvc  *ImageService
	statSvc *StatService

	defaultImage []byte
}

var ErrUnauthorized = errors.New("unauthorized")
var DeleteResponseOK = `{"status": 200, "error": ""}`

func (svc HttpService) Id() string {
	return "http"
}

func (svc *HttpService) Configure(ctx *context.Context) error {
	port := os.Getenv("HTTP_PORT")
	portFlag, err := strconv.Atoi(port)
	if err != nil {
		return err
	}

	svc.Port = portFlag

	svc.defaultImage, err = os.ReadFile("./docs/failed_image.jpg")
	if err != nil {
		return err
	}

	return svc.DefaultService.Configure(ctx)
}

func (svc *HttpService) Start() error {
	svc.imgSvc = svc.DefaultService(IMG_SVC).(*ImageService)
	svc.statSvc = svc.DefaultService(STAT_SVC).(*StatService)

	r := gin.Default()

	r.Use(gin.Recovery())

	config := cors.DefaultConfig()
	// config.AllowAllOrigins = true. This may not be suitable for production environments as it could lead to security vulnerabilities
	config.AllowAllOrigins = true
	config.AllowCredentials = true
	config.AddAllowHeaders("Authorization")
	r.Use(cors.New(config))

	//r.Static("static", "static")
	//r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerfiles.Handler))

	//Validation endpoints
	r.GET("/ping", svc.ping)
	r.GET("/stats", svc.stats)

	v1 := r.Group("/v1")
	//docs.SwaggerInfo.BasePath = "/v1"

	v1.GET("tokens/:id", svc.showNFT)
	v1.GET("tokens/:id/image", svc.showNFTImage)
	v1.GET("tokens/:id/image.gif", svc.showNFTImage)
	v1.GET("tokens/:id/image.png", svc.showNFTImage)
	v1.GET("tokens/:id/image.jpg", svc.showNFTImage)
	v1.GET("tokens/:id/image.jpeg", svc.showNFTImage)
	v1.GET("tokens/:id/media", svc.showNFTMedia)

	v1.GET("nfts/:id", svc.showNFT)
	v1.GET("nfts/:id/image", svc.showNFTImage)
	v1.GET("nfts/:id/image.gif", svc.showNFTImage)
	v1.GET("nfts/:id/image.png", svc.showNFTImage)
	v1.GET("nfts/:id/image.jpg", svc.showNFTImage)
	v1.GET("nfts/:id/image.jpeg", svc.showNFTImage)
	v1.GET("nfts/:id/media", svc.showNFTMedia)

	r.NoRoute(func(c *gin.Context) {
		c.JSON(404, gin.H{"code": "PAGE_NOT_FOUND", "message": "Page not found"})
	})

	return r.Run(fmt.Sprintf(":%v", svc.Port))
}

type Pong struct {
	Message string `json:"message"`
}

// @Summary Ping liquify service
// @Accept  json
// @Produce json
// @Router /ping [get]
func (svc *HttpService) ping(c *gin.Context) {
	c.JSON(200, gin.H{
		"message": "pong",
	})
}

// @Summary Ping liquify service
// @Accept  json
// @Produce json
// @Router /stats [get]
func (svc *HttpService) stats(c *gin.Context) {
	stats, err := svc.statSvc.ServiceStats()
	if err != nil {
		svc.paramErr(c, err)
		return
	}

	c.JSON(200, stats)
}

// @Summary Ping liquify service
// @Accept  json
// @Produce json
// @Router /nfts/{id} [get]
func (svc *HttpService) showNFT(c *gin.Context) {
	svc.statSvc.IncrementMediaRequests()

	skipCache, _ := strconv.ParseBool(c.DefaultQuery("nocache", ""))
	// Hard-coded Magic Numbers
	if skipCache || rand.Intn(1000) == 1 {
		if err := svc.imgSvc.ClearCache(c.Param("id")); err != nil {
			svc.paramErr(c, err)
			return
		}
	}

	media, err := svc.imgSvc.Media(c.Param("id"), skipCache)
	if err != nil {
		svc.paramErr(c, err)
		return
	}

	c.Header("Cache-Control", "public, max-age=172800")
	c.Header("Expires", time.Now().AddDate(0, 0, 2).Format(http.TimeFormat))

	c.JSON(200, media)
}

// @Summary Ping liquify service
// @Accept  json
// @Produce json
// @Router/nfts/{id}/image [get]
func (svc *HttpService) showNFTImage(c *gin.Context) {
	svc.statSvc.IncrementImageFileRequests()
	err := svc.imgSvc.ImageFile(c, c.Param("id"))
	// When an error occurs, a 200 status code is returned along with the default image which is misleading since 200 status code indicates success
	if err != nil {
		svc.mediaError(c, err)
		return
	}
}

// @Summary Ping liquify service
// @Accept  json
// @Produce json
// @Router /nfts/{id}/media [get]
func (svc *HttpService) showNFTMedia(c *gin.Context) {
	svc.statSvc.IncrementMediaFileRequests()
	err := svc.imgSvc.MediaFile(c, c.Param("id"))
	// When an error occurs, a 200 status code is returned along with the default image which is misleading since 200 status code indicates success
	if err != nil {
		svc.mediaError(c, err)
		return
	}
}

func (svc *HttpService) paramErr(c *gin.Context, err error) {
	c.JSON(400, gin.H{
		"error": err.Error(),
	})
}

// TODO Replace with placeholder image
func (svc *HttpService) mediaError(c *gin.Context, err error) {
	log.Printf("Media Err: %s", err)
	// Cache Control Header Typo: In the mediaError function, the Cache-Control header has a typo: max=age=60 should be max-age=60
	c.Header("Cache-Control", "public, max=age=60") //Stop flooding
	c.Data(http.StatusNotFound, "image/jpeg", svc.defaultImage) // Return 404 instead of 200
	//c.JSON(200, gin.H{
	//	"error": err.Error(),
	//})
}
