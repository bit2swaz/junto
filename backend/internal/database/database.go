package database

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/joho/godotenv/autoload" // Autoload .env file
	"github.com/redis/go-redis/v9"
)

type Service interface {
	Health() map[string]string
	Close()
	GetPool() *pgxpool.Pool
	GetRedis() *redis.Client
	CreateUser(ctx context.Context, email, passwordHash string) (*User, error)
	GetUserByEmail(ctx context.Context, email string) (*User, error)
	CreateCouple(ctx context.Context, user1ID, user2ID int64) (*Couple, error)
}

type service struct {
	db    *pgxpool.Pool
	redis *redis.Client
}

var (
	database   = os.Getenv("DB_DATABASE")
	password   = os.Getenv("DB_PASSWORD")
	username   = os.Getenv("DB_USERNAME")
	port       = os.Getenv("DB_PORT")
	host       = os.Getenv("DB_HOST")
	dbInstance *service
)

func NewService() (Service, error) {
	// Construct the connection string
	// If DB_URL is provided, use it, otherwise build from components
	connStr := os.Getenv("DATABASE_URL")
	if connStr == "" {
		connStr = fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable", username, password, host, port, database)
	}

	config, err := pgxpool.ParseConfig(connStr)
	if err != nil {
		return nil, fmt.Errorf("unable to parse database config: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		return nil, fmt.Errorf("unable to create connection pool: %v", err)
	}

	// Ping the database to ensure connection is established
	if err := pool.Ping(ctx); err != nil {
		return nil, fmt.Errorf("unable to ping database: %v", err)
	}

	// Initialize Redis
	redisAddr := os.Getenv("REDIS_ADDR")
	if redisAddr == "" {
		redisAddr = "localhost:6379"
	}
	rdb := redis.NewClient(&redis.Options{
		Addr: redisAddr,
	})

	if err := rdb.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("unable to ping redis: %v", err)
	}

	log.Println("Connected to database and redis")
	return &service{db: pool, redis: rdb}, nil
}

func (s *service) Health() map[string]string {
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	stats := make(map[string]string)

	err := s.db.Ping(ctx)
	if err != nil {
		stats["db_status"] = "down"
		stats["db_error"] = fmt.Sprintf("db down: %v", err)
	} else {
		stats["db_status"] = "up"
	}

	err = s.redis.Ping(ctx).Err()
	if err != nil {
		stats["redis_status"] = "down"
		stats["redis_error"] = fmt.Sprintf("redis down: %v", err)
	} else {
		stats["redis_status"] = "up"
	}

	return stats
}

func (s *service) Close() {
	s.db.Close()
	s.redis.Close()
}

func (s *service) GetPool() *pgxpool.Pool {
	return s.db
}

func (s *service) GetRedis() *redis.Client {
	return s.redis
}
