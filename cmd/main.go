package main

import (
	"fmt"
	"log"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	_ "github.com/joho/godotenv/autoload"
	_ "github.com/lib/pq"
	"github.com/robfig/cron/v3"
	"go.uber.org/zap"

	"bone_appetit_suscription/internal/config"
	"bone_appetit_suscription/internal/handlers"
	"bone_appetit_suscription/internal/jobs"
	"bone_appetit_suscription/internal/routers"
	"bone_appetit_suscription/internal/services"
	"bone_appetit_suscription/pkg/db"
	"bone_appetit_suscription/pkg/foodapi"
	"bone_appetit_suscription/pkg/logs"
	"bone_appetit_suscription/pkg/mailgun"
	"bone_appetit_suscription/pkg/shopify"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("loading config: %v", err)
	}

	if cfg.Port == "" {
		cfg.Port = "8080"
	}
	if cfg.Debug == "" {
		cfg.Debug = "0"
	}

	logger := logs.NewZapLogger()
	defer func() {
		if err := logger.Sync(); err != nil {
			fmt.Printf("error syncing logger: %v\n", err)
		}
	}()

	sslmode := cfg.SSLMode
	fmt.Printf("sslmode -> %s\n", sslmode)
	if len(sslmode) > 0 {
		sslmode = "sslmode=" + sslmode
	}

	// connect the database
	connStr := fmt.Sprintf("host=%s port=%s user=%s "+
		"password=%s dbname=%s %s",
		cfg.DBHost, cfg.DBPort, cfg.DBUser, cfg.DBPassword, cfg.DBName, sslmode)
	// gorm connect
	gormDB, err := db.NewDBSQLHandler(connStr)
	if err != nil {
		logger.Error(err.Error(), zap.Any("host", cfg.DBHost), zap.Any("port", cfg.DBPort), zap.Any("user", cfg.DBUser), zap.Any("dbname", cfg.DBName))
	}

	db, err := gormDB.DB()
	if err != nil {
		logger.Error(err.Error(), zap.Any("host", cfg.DBHost), zap.Any("port", cfg.DBPort), zap.Any("user", cfg.DBUser), zap.Any("dbname", cfg.DBName))
	}
	defer func() {
		if err := db.Close(); err != nil {
			fmt.Printf("error db body: %v\n", err)
		}
	}()

	loc, err := time.LoadLocation("America/Caracas")
	if err != nil {
		logger.Fatal("could not load Venezuela time zone", zap.Error(err))
	}

	muClient := mailgun.NewClient(cfg.MailgunAPIKey)

	// Initialize resources
	muRepository := mailgun.NewRepository(muClient, cfg.MailgunDomain, cfg.MailgunSender, logger)

	router := gin.Default()
	router.Use(cors.Default())

	router.GET("/healthz", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"status": "OK",
		})
	})

	// Initialize API clients / repositories
	shopifyCliente := shopify.NewRepository(
		cfg.ShopifyStoreName, cfg.ShopifyAPIVersion, cfg.ShopifyAdminToken, logger,
	)

	foodAPIClient := foodapi.NewClient(cfg.FoodAPIURL, cfg.FoodAPITimeout, logger)

	// Initialize services
	orderService := services.NewOrderService(gormDB, shopifyCliente, muRepository, foodAPIClient, loc, logger)
	webhookService := services.NewWebhookService(gormDB, logger, shopifyCliente, loc)

	// Initialize handlers
	orderHandler := handlers.NewOrderHandler(orderService)
	webhookHandler := handlers.NewWebhookHandler(webhookService, orderService, logger)

	// Initialize routes
	orderRouter := routers.NewOrderRouter(*orderHandler)
	webhookRouter := routers.NewWebhookRoutes(webhookHandler)

	orderRouter.SetRouter(router)
	webhookRouter.SetRouter(router, cfg.ShopifyHMACSecret)

	// Init Crons
	// init config cron
	c := cron.New(
		cron.WithSeconds(),
		cron.WithLocation(loc),
	)

	jobHandler := jobs.NewJobHandler(orderService, logger)
	jobHandler.StartWorkerPool(4)
	// Add TIIE job -> RUN | 09:00am | ALL DAYS |
	// _, err = c.AddFunc("0 50 8 * * *", jobHandler.HandleScheduledOrders)
	// if err != nil {
	// 	logger.Fatal("error adding job HandleScheduledOrders to cron", zap.Error(err))
	// }

	// Add TIIE job -> RUN | 09:00am | ALL DAYS |
	_, err = c.AddFunc("0 15 8 * * *", jobHandler.HandleScheduledOrders)
	if err != nil {
		logger.Fatal("error adding job HandleScheduledOrders to cron", zap.Error(err))
	}

	// Add TIIE job -> RUN | 1:00pm | ALL DAYS |
	// _, err = c.AddFunc("0 0 11 * * *", jobHandler.HandleReminder)
	// if err != nil {
	// 	logger.Fatal("error adding job HandleReminder to cron", zap.Error(err))
	// }

	// init cron jobs
	if cfg.Debug != "1" {
		c.Start()
		defer c.Stop()
	} else {
		go func() {
			logger.Info("Starting")
			jobHandler.HandleScheduledOrders()
			logger.Info("Finisheds")
		}()
	}

	// jobHandler.HandleScheduledOrders()

	if err := router.Run(":" + cfg.Port); err != nil {
		log.Fatalf("failed to run server: %v", err)
	}
}
