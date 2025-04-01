package config

import (
	"context"
	"encoding/base64"
	"errors"
	"os"

	firebase "firebase.google.com/go/v4"
	"firebase.google.com/go/v4/auth"
	"github.com/labstack/gommon/log"
	"google.golang.org/api/option"
)

var (
	FirebaseApp  *firebase.App
	FirebaseAuth *auth.Client
)

// InitFirebase initializes the Firebase app and auth client
func InitFirebase() error {
	// Get Firebase credentials from environment variable
	firebaseConfig := os.Getenv("FIREBASE_CONFIG_DATA")
	if firebaseConfig == "" {
		return errors.New("FIREBASE_CONFIG_DATA environment variable not set")
	}

	// its a base64 encoded json
	decodedConfig, err := base64.StdEncoding.DecodeString(firebaseConfig)
	if err != nil {
		return err
	}

	// Initialize Firebase app with credentials
	opt := option.WithCredentialsJSON(decodedConfig)

	app, err := firebase.NewApp(context.Background(), nil, opt)
	if err != nil {
		return err
	}
	FirebaseApp = app

	// Initialize Firebase Auth client
	auth, err := app.Auth(context.Background())
	if err != nil {
		log.Info("Failed to initialize Firebase auth: %v", err)
		return err
	}
	FirebaseAuth = auth

	return nil
}
