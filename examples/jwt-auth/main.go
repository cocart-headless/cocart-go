// Example: JWT Authentication with CoCart SDK
//
// This example demonstrates JWT login, token management,
// auto-refresh, and the guest-to-customer cart transfer flow.
package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	cocart "github.com/cocart-headless/cocart-sdk-go"
)

func main() {
	ctx := context.Background()

	// Create a client with storage for persisting cart keys and tokens
	client := cocart.NewClient("https://your-store.com",
		cocart.WithStorage(cocart.NewMemoryStorage()),
	)
	session := client.Session()

	// --- Guest Phase ---

	// Initialize a guest cart session
	cartKey, err := session.InitializeCart(ctx)
	if err != nil {
		log.Fatal("Failed to initialize cart:", err)
	}
	fmt.Println("Guest cart key:", cartKey)

	// Add items as a guest
	_, err = client.Cart().AddItem(ctx, 123, 2)
	if err != nil {
		log.Fatal("Failed to add item:", err)
	}
	fmt.Println("Added item to guest cart")

	// --- Login Phase ---

	// Login via JWT (mergeCart=true transfers guest items to customer cart)
	loginResp, err := session.LoginWithJWT(ctx, "customer@email.com", "password", true)
	if err != nil {
		log.Fatal("Failed to login:", err)
	}
	fmt.Println("Logged in as:", loginResp.Get("display_name", ""))

	// Check authentication status
	fmt.Println("Authenticated:", session.IsAuthenticated())
	fmt.Println("Guest:", session.IsGuest())

	// --- Authenticated Phase ---

	// Access cart as authenticated customer (guest items were merged)
	cart, err := client.Cart().Get(ctx, nil)
	if err != nil {
		log.Fatal("Failed to get cart:", err)
	}
	fmt.Println("Cart items after login:", cart.GetItemCount())

	// JWT token management
	jwt := client.JWT()
	fmt.Println("Has tokens:", jwt.HasTokens())
	fmt.Println("Token expired:", jwt.IsTokenExpired())

	expiry := jwt.GetTokenExpiry()
	if expiry != nil {
		fmt.Println("Token expiry:", expiry.Format(time.RFC3339))
	}

	// Manually refresh token if needed
	_, err = jwt.Refresh(ctx)
	if err != nil {
		// Handle refresh error
		var authErr *cocart.AuthenticationError
		if errors.As(err, &authErr) {
			fmt.Println("Refresh failed, need to re-login:", authErr.Message)
		}
	}

	// The SDK automatically refreshes tokens on 401 responses
	// if a refresh token is available. No extra code needed.

	// --- Logout Phase ---

	err = session.Logout(ctx)
	if err != nil {
		log.Fatal("Failed to logout:", err)
	}
	fmt.Println("Logged out")
	fmt.Println("Guest after logout:", session.IsGuest())
}
