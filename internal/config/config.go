package config

import "os"

type Config struct {
	MongoURI	string
	Port		string
	JWTSecret	string
	Environment string
	BackendURL	string
}

func Load() *Config {
	port := os.Getenv("PORT")
	if port == "" {
		port  = "8080"
	}
	return &Config{
		MongoURI:		os.Getenv("MONGODB_URI"),
		Port: 			port,
		JWTSecret: 		os.Getenv("JWT_SECRET"),
		Environment:	os.Getenv("ENVIRONMENT"),
		BackendURL: 	os.Getenv("BACKEND_URL"),
	}
}