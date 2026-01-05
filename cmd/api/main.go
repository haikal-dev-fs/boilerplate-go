package main

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/joho/godotenv"
	mw "github.com/username/fms-api/internal/middleware"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	authHandler "github.com/username/fms-api/internal/auth"
	deviceHandler "github.com/username/fms-api/internal/device"
	orgHandler "github.com/username/fms-api/internal/organization"
	userHandler "github.com/username/fms-api/internal/user"

	"github.com/username/fms-api/internal/alert"
	"github.com/username/fms-api/internal/auth"
	"github.com/username/fms-api/internal/trip"
	"github.com/username/fms-api/internal/vehicle"
)

func main() {
	// 0. Load environment variables dari .env file
	if err := godotenv.Load(); err != nil {
		log.Printf("Warning: gagal memuat file .env: %v", err)
	}

	// 1. Siapkan URL database
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgres://fms_user:fms_password@localhost:5432/fms_db?sslmode=disable"
	}

	// 2. Jalankan migration dulu
	m, err := migrate.New("file://migrations", dbURL)
	if err != nil {
		log.Fatalf("Gagal membuat instance migrasi: %v", err)
	}

	err = m.Up()
	if err != nil {
		if err == migrate.ErrNoChange {
			fmt.Println("ℹ️  Tidak ada migration baru, schema sudah up to date")
		} else {
			log.Fatalf("gagal menjalankan migration: %v", err)
		}
	} else {
		fmt.Println("✅ Migration berhasil dijalankan")
	}

	//  3. Buka koneksi database dengan GORM
	gormDB, err := gorm.Open(postgres.Open(dbURL), &gorm.Config{})
	if err != nil {
		log.Fatalf("Gagal membuka koneksi database dengan GORM: %v", err)
	}
	fmt.Println("✅ Koneksi database dengan GORM berhasil dibuka")

	// 4. Inisialisasi Gin
	// gin.Default() sudah include logger + recovery middleware
	router := gin.Default()
	// Add CORS middleware so the frontend (served from file:// or other origin)
	// can call the API during development.
	router.Use(mw.CORSMiddleware())

	// 5. Route /health
	router.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"status":    "ok",
			"timestamp": time.Now().Format(time.RFC3339),
		})
	})

	// 6. Route untuk Swagger (serve file statis dari folder ./swagger)
	// Ini artinya:
	// /swagger/index.html -> swagger/index.html
	// /swagger/openapi.yaml -> swagger/openapi.yaml
	router.Static("/swagger", "./swagger")

	// auth login (tidak pakai middleware)
	authH := authHandler.NewHandler(gormDB)
	authH.RegisterRoutes(router)

	// 7. Group API yang butuh auth
	api := router.Group("/api")
	api.Use(auth.AuthMiddleware())

	// Daftarkan handler ke group ini
	vehicleHandler := vehicle.NewHandler(gormDB)
	vehicleHandler.RegisterRoutes(api)

	userH := userHandler.NewHandler(gormDB)
	userH.RegisterRoutes(api)

	tripHandler := trip.NewHandler(gormDB)
	tripHandler.RegisterRoutes(api)

	alertHandler := alert.NewHandler(gormDB)
	alertHandler.RegisterRoutes(api)

	admin := router.Group("/admin")
	admin.Use(auth.AuthMiddleware())

	orgH := orgHandler.NewHandler(gormDB)
	orgH.RegisterAdminRoutes(admin)

	devH := deviceHandler.NewHandler(gormDB)
	devH.RegisterAdminRoutes(admin)

	// 8. Start server
	addr := ":8080"
	fmt.Println("🚀 Server berjalan di http://localhost" + addr)
	fmt.Println("📘 Swagger UI di: http://localhost" + addr + "/swagger/")

	if err := router.Run(addr); err != nil {
		log.Fatalf("gagal menjalankan HTTP server: %v", err)
	}
}
