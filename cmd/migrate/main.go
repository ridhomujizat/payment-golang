package main

import (
	"context"
	config "go-boilerplate/configs"
	database "go-boilerplate/internal/pkg/db"
	"go-boilerplate/internal/pkg/logger"
	"go-boilerplate/internal/pkg/rabbitmq"
)

func main() {
	logger.Setup()
	env, err := config.GetEnv()
	if err != nil {
		logger.Error.Println("Error getting environment", err)
		panic(err)
	}
	_, cancel := context.WithCancel(context.Background())
	defer cancel() // Ensure context is canceled when done

	// Setup Database
	db, err := setupDB(env)
	if err != nil {
		logger.Error.Println("Error setting up Database", err)
		return
	}

	defer func() {
		if db != nil {
			db.Close()
		}
	}()

	err = db.RunMigrations()
	if err != nil {
		logger.Error.Println("Error running migrations", err)
		return
	}

	logger.Info.Println("Migrations completed successfully")
}

func setupDB(env *config.Config) (*database.Database, error) {
	return database.Setup(&database.Config{
		Host:     env.DBHost,
		Port:     env.DBPort,
		User:     env.DBUser,
		Password: env.DBPass,
		Database: env.DBName,
		SSLMode:  "disable",
		Driver:   "postgres",
	})
}

func setupRabbitMQ(ctx context.Context, env *config.Config) (*rabbitmq.ConnectionManager, error) {
	return rabbitmq.NewConnectionManager(ctx, &rabbitmq.Config{
		Username: env.RabbitUser,
		Password: env.RabbitPass,
		Host:     env.RabbitHost,
		Port:     env.RabbitPort,
	})
}
