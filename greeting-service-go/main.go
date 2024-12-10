/*
 * Copyright (c) 2023, WSO2 LLC. (https://www.wso2.com/) All Rights Reserved.
 *
 * WSO2 LLC. licenses this file to you under the Apache License,
 * Version 2.0 (the "License"); you may not use this file except
 * in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing,
 * software distributed under the License is distributed on an
 * "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
 * KIND, either express or implied. See the License for the
 * specific language governing permissions and limitations
 * under the License.
 */

package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
        "io"
        "golang.org/x/oauth2/clientcredentials"
)

func main() {

	serverMux := http.NewServeMux()
	serverMux.HandleFunc("/greeter/greet", greet)

	serverPort := 9090
	server := http.Server{
		Addr:    fmt.Sprintf(":%d", serverPort),
		Handler: serverMux,
	}
	go func() {
		log.Printf("Starting HTTP Greeter on port %d\n", serverPort)
		if err := server.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("HTTP ListenAndServe error: %v", err)
		}
		log.Println("HTTP server stopped serving new requests.")
	}()

	stopCh := make(chan os.Signal, 1)
	signal.Notify(stopCh, syscall.SIGINT, syscall.SIGTERM)
	<-stopCh // Wait for shutdown signal

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	log.Println("Shutting down the server...")
	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Fatalf("HTTP shutdown error: %v", err)
	}
	log.Println("Shutdown complete.")
}

func greet(w http.ResponseWriter, r *http.Request) {
	// Read environment variables
	serviceURL := os.Getenv("CHOREO_TEYSYD_SERVICEURL")+"/greeting"
	tokenURL := os.Getenv("CHOREO_TEYSYD_TOKENURL")
	clientSecret := os.Getenv("CHOREO_TEYSYD_CONSUMERSECRET")
	clientID := os.Getenv("CHOREO_TEYSYD_CONSUMERKEY")
	
	// Log the client ID for debugging
	fmt.Printf("Client ID: %s\n", clientID)
	fmt.Printf("serviceURL: %s\n", serviceURL)

	// // Prepare a JSON response with the environment variables
	// envVars := map[string]string{
	// 	"SVC_URL":         serviceURL,
	// 	"TOKEN_URL":       tokenURL,
	// 	"CONSUMER_SECRET": clientSecret,
	// 	"CONSUMER_KEY":    clientID,
	// }

	// // Set the content type to JSON and write the response
	// w.Header().Set("Content-Type", "application/json")
	// if err := json.NewEncoder(w).Encode(envVars); err != nil {
	// 	http.Error(w, fmt.Sprintf("Failed to encode response: %v", err), http.StatusInternalServerError)
	// }
	
	var clientCredsConfig = clientcredentials.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		TokenURL:     tokenURL,
	}

	client := clientCredsConfig.Client(context.Background())
	response, err := client.Get(serviceURL)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error making request: %v", err), http.StatusInternalServerError)
		return
	}
	defer response.Body.Close()
	// Check the response status code
	if response.StatusCode != http.StatusOK {
		http.Error(w, fmt.Sprintf("Server returned non-200 status: %d %s", response.StatusCode, response.Status), response.StatusCode)
		return
	}
	// Copy the response body directly to the client
	_, err = io.Copy(w, response.Body)
	if err != nil {
		// If there's an error writing the response body to the client, log it
		log.Printf("Error writing response body to client: %v\n", err)
	}
}
