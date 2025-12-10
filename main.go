package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// Product represents a bakery item
type Product struct {
	ID          int     `json:"id"`
	Name        string  `json:"name"`
	Description string  `json:"description"`
	Price       float64 `json:"price"`
	Image       string  `json:"image"`
	Category    string  `json:"category"`
}

// OrderItem represents an item in an order
type OrderItem struct {
	ProductID int `json:"productId"`
	Quantity  int `json:"quantity"`
}

// Order represents a customer order
type Order struct {
	ID        primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	OrderID   int                `bson:"orderId" json:"orderId"`
	Customer  Customer           `bson:"customer" json:"customer"`
	Items     []OrderItem        `bson:"items" json:"items"`
	Total     float64            `bson:"total" json:"total"`
	Status    string             `bson:"status" json:"status"`
	CreatedAt time.Time          `bson:"createdAt" json:"createdAt"`
}

// Customer represents customer information
type Customer struct {
	Name    string `json:"name"`
	Email   string `json:"email"`
	Phone   string `json:"phone"`
	Address string `json:"address"`
}

// Global variables
var (
	products            []Product
	productsMu          sync.RWMutex
	mongoClient         *mongo.Client
	ordersCollection    *mongo.Collection
	deliveredCollection *mongo.Collection
	adminUsername       string
	adminPassword       string
	activeSessions      = make(map[string]time.Time)
	sessionsMu          sync.RWMutex
)

func init() {
	// Initialize with sample products
	products = []Product{
		{ID: 1, Name: "Chocolate Chip Cookies", Description: "Freshly baked cookies with premium chocolate chips", Price: 8.99, Image: "🍪", Category: "Cookies"},
		{ID: 2, Name: "Blueberry Muffins", Description: "Moist muffins bursting with fresh blueberries", Price: 6.99, Image: "🧁", Category: "Muffins"},
		{ID: 3, Name: "Croissant", Description: "Buttery, flaky French croissant", Price: 4.99, Image: "🥐", Category: "Pastries"},
		{ID: 4, Name: "Chocolate Cake", Description: "Rich chocolate layer cake with buttercream frosting", Price: 24.99, Image: "🎂", Category: "Cakes"},
		{ID: 5, Name: "Apple Pie", Description: "Homemade apple pie with cinnamon", Price: 18.99, Image: "🥧", Category: "Pies"},
		{ID: 6, Name: "Bagels", Description: "Fresh New York style bagels (pack of 6)", Price: 7.99, Image: "🥯", Category: "Breads"},
		{ID: 7, Name: "Cinnamon Roll", Description: "Warm cinnamon rolls with cream cheese glaze", Price: 5.99, Image: "🍩", Category: "Pastries"},
		{ID: 8, Name: "Strawberry Tart", Description: "Delicate tart with fresh strawberries", Price: 12.99, Image: "🍓", Category: "Tarts"},
	}
}

func main() {
	// Connect to MongoDB
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	mongoURI := "mongodb://localhost:27017"
	if uri := getEnv("MONGODB_URI", ""); uri != "" {
		mongoURI = uri
	}

	client, err := mongo.Connect(ctx, options.Client().ApplyURI(mongoURI))
	if err != nil {
		log.Fatal("Failed to connect to MongoDB:", err)
	}

	// Test connection
	err = client.Ping(ctx, nil)
	if err != nil {
		log.Fatal("Failed to ping MongoDB:", err)
	}

	mongoClient = client
	ordersCollection = client.Database("sububakery").Collection("orders")
	deliveredCollection = client.Database("sububakery").Collection("delivered")

	// Get admin credentials from environment or use defaults
	adminUsername = getEnv("ADMIN_USERNAME", "admin")
	adminPassword = getEnv("ADMIN_PASSWORD", "subu369")

	// Clean up expired sessions periodically
	go cleanupSessions()

	fmt.Println("Connected to MongoDB successfully")
	fmt.Printf("Admin credentials: username='%s', password='%s'\n", adminUsername, adminPassword)

	router := gin.Default()

	// CORS middleware - configured for ngrok
	config := cors.DefaultConfig()
	config.AllowAllOrigins = true
	config.AllowMethods = []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"}
	config.AllowHeaders = []string{"Origin", "Content-Type", "Accept", "Authorization", "X-Requested-With"}
	config.AllowCredentials = true
	config.ExposeHeaders = []string{"Content-Length", "Content-Type"}
	router.Use(cors.New(config))

	// Serve static files
	router.Static("/static", "./static")
	router.LoadHTMLGlob("templates/*")

	// Routes
	router.GET("/", func(c *gin.Context) {
		c.HTML(http.StatusOK, "index.html", nil)
	})

	router.GET("/orders", func(c *gin.Context) {
		c.HTML(http.StatusOK, "orders.html", nil)
	})

	// Public API routes
	api := router.Group("/api")
	{
		api.GET("/products", getProducts)
		api.GET("/products/:id", getProduct)
		api.POST("/orders", createOrder)

		// Authentication routes
		api.POST("/auth/login", handleLogin)
		api.POST("/auth/logout", handleLogout)
		api.GET("/auth/check", checkAuth)

		// Protected routes (require authentication)
		protected := api.Group("")
		protected.Use(requireAuth())
		{
			protected.GET("/orders", getOrders)
			protected.GET("/orders/:id", getOrder)
			protected.POST("/orders/:id/deliver", markOrderDelivered)
			protected.GET("/delivered", getDeliveredOrders)
		}
	}

	// Start server
	fmt.Println("Subu Bakery server starting on :8080")
	log.Fatal(router.Run(":8080"))
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// getProducts returns all products
func getProducts(c *gin.Context) {
	productsMu.RLock()
	defer productsMu.RUnlock()
	c.JSON(http.StatusOK, products)
}

// getProduct returns a single product by ID
func getProduct(c *gin.Context) {
	id := c.Param("id")
	productsMu.RLock()
	defer productsMu.RUnlock()

	for _, product := range products {
		if fmt.Sprintf("%d", product.ID) == id {
			c.JSON(http.StatusOK, product)
			return
		}
	}
	c.JSON(http.StatusNotFound, gin.H{"error": "Product not found"})
}

// createOrder creates a new order
func createOrder(c *gin.Context) {
	var orderReq struct {
		Customer Customer    `json:"customer"`
		Items    []OrderItem `json:"items"`
	}

	if err := c.ShouldBindJSON(&orderReq); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validate order
	if len(orderReq.Items) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Order must contain at least one item"})
		return
	}

	// Calculate total
	var total float64
	productsMu.RLock()
	for _, item := range orderReq.Items {
		for _, product := range products {
			if product.ID == item.ProductID {
				total += product.Price * float64(item.Quantity)
				break
			}
		}
	}
	productsMu.RUnlock()

	// Get next order ID from database
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	nextOrderID, err := getNextOrderID(ctx)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate order ID"})
		return
	}

	// Create order
	order := Order{
		ID:        primitive.NewObjectID(),
		OrderID:   nextOrderID,
		Customer:  orderReq.Customer,
		Items:     orderReq.Items,
		Total:     total,
		Status:    "pending",
		CreatedAt: time.Now(),
	}

	// Insert into MongoDB
	_, err = ordersCollection.InsertOne(ctx, order)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save order"})
		return
	}

	c.JSON(http.StatusCreated, order)
}

// getOrders returns all orders
func getOrders(c *gin.Context) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Find all orders and sort by createdAt descending (newest first)
	findOptions := options.Find()
	findOptions.SetSort(bson.D{primitive.E{Key: "createdAt", Value: -1}})

	cursor, err := ordersCollection.Find(ctx, bson.M{}, findOptions)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch orders"})
		return
	}
	defer cursor.Close(ctx)

	var orders []Order
	if err = cursor.All(ctx, &orders); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to decode orders"})
		return
	}

	c.JSON(http.StatusOK, orders)
}

// getOrder returns a single order by ID
func getOrder(c *gin.Context) {
	id := c.Param("id")

	// Try to parse as ObjectID first
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		// If not ObjectID, try as orderId (int)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid order ID format"})
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var order Order
	err = ordersCollection.FindOne(ctx, bson.M{"_id": objectID}).Decode(&order)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			c.JSON(http.StatusNotFound, gin.H{"error": "Order not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch order"})
		return
	}

	c.JSON(http.StatusOK, order)
}

// markOrderDelivered moves an order from orders collection to delivered collection
func markOrderDelivered(c *gin.Context) {
	id := c.Param("id")

	// Try to parse as ObjectID
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid order ID format"})
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Find the order in orders collection
	var order Order
	err = ordersCollection.FindOne(ctx, bson.M{"_id": objectID}).Decode(&order)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			c.JSON(http.StatusNotFound, gin.H{"error": "Order not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch order"})
		return
	}

	// Update order status and add delivered timestamp
	order.Status = "delivered"
	deliveredAt := time.Now()

	// Create delivered order document with additional tracking info
	deliveredOrder := bson.M{
		"_id":         order.ID,
		"orderId":     order.OrderID,
		"customer":    order.Customer,
		"items":       order.Items,
		"total":       order.Total,
		"status":      "delivered",
		"createdAt":   order.CreatedAt,
		"deliveredAt": deliveredAt,
	}

	// Insert into delivered collection
	_, err = deliveredCollection.InsertOne(ctx, deliveredOrder)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save delivered order"})
		return
	}

	// Delete from orders collection
	_, err = ordersCollection.DeleteOne(ctx, bson.M{"_id": objectID})
	if err != nil {
		// If delete fails, try to remove from delivered collection to maintain consistency
		deliveredCollection.DeleteOne(ctx, bson.M{"_id": objectID})
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to remove order from active orders"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":     "Order marked as delivered",
		"orderId":     order.OrderID,
		"deliveredAt": deliveredAt,
	})
}

// getDeliveredOrders returns all delivered orders
func getDeliveredOrders(c *gin.Context) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Find all delivered orders and sort by deliveredAt descending (newest first)
	findOptions := options.Find()
	findOptions.SetSort(bson.D{primitive.E{Key: "deliveredAt", Value: -1}})

	cursor, err := deliveredCollection.Find(ctx, bson.M{}, findOptions)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch delivered orders"})
		return
	}
	defer cursor.Close(ctx)

	var deliveredOrders []bson.M
	if err = cursor.All(ctx, &deliveredOrders); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to decode delivered orders"})
		return
	}

	c.JSON(http.StatusOK, deliveredOrders)
}

// Authentication middleware
func requireAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		token := c.GetHeader("Authorization")
		if token == "" {
			// Try to get from cookie
			cookie, err := c.Cookie("auth_token")
			if err == nil {
				token = cookie
			}
		}

		if token == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication required"})
			c.Abort()
			return
		}

		// Remove "Bearer " prefix if present
		if len(token) > 7 && token[:7] == "Bearer " {
			token = token[7:]
		}

		sessionsMu.RLock()
		expiry, exists := activeSessions[token]
		sessionsMu.RUnlock()

		if !exists || time.Now().After(expiry) {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or expired session"})
			c.Abort()
			return
		}

		c.Next()
	}
}

// handleLogin processes login requests
func handleLogin(c *gin.Context) {
	var loginReq struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}

	if err := c.ShouldBindJSON(&loginReq); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	// Check credentials
	if loginReq.Username != adminUsername || loginReq.Password != adminPassword {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
		return
	}

	// Generate session token
	token, err := generateToken()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create session"})
		return
	}

	// Store session (valid for 24 hours)
	sessionsMu.Lock()
	activeSessions[token] = time.Now().Add(24 * time.Hour)
	sessionsMu.Unlock()

	// Set cookie - configured for ngrok
	// Check if request is from ngrok (HTTPS) or localhost
	isSecure := c.Request.TLS != nil || c.GetHeader("X-Forwarded-Proto") == "https"
	setCookieWithSameSite(c, "auth_token", token, 86400, "/", "", isSecure, true)

	c.JSON(http.StatusOK, gin.H{
		"token":     token,
		"message":   "Login successful",
		"expiresIn": 86400,
	})
}

// handleLogout processes logout requests
func handleLogout(c *gin.Context) {
	token := c.GetHeader("Authorization")
	if len(token) > 7 && token[:7] == "Bearer " {
		token = token[7:]
	} else {
		cookie, err := c.Cookie("auth_token")
		if err == nil {
			token = cookie
		}
	}

	if token != "" {
		sessionsMu.Lock()
		delete(activeSessions, token)
		sessionsMu.Unlock()
	}

	// Clear cookie - configured for ngrok
	isSecure := c.Request.TLS != nil || c.GetHeader("X-Forwarded-Proto") == "https"
	setCookieWithSameSite(c, "auth_token", "", -1, "/", "", isSecure, true)
	c.JSON(http.StatusOK, gin.H{"message": "Logged out successfully"})
}

// checkAuth checks if user is authenticated
func checkAuth(c *gin.Context) {
	token := c.GetHeader("Authorization")
	if len(token) > 7 && token[:7] == "Bearer " {
		token = token[7:]
	} else {
		cookie, err := c.Cookie("auth_token")
		if err == nil {
			token = cookie
		}
	}

	if token == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"authenticated": false})
		return
	}

	sessionsMu.RLock()
	expiry, exists := activeSessions[token]
	sessionsMu.RUnlock()

	if exists && time.Now().Before(expiry) {
		c.JSON(http.StatusOK, gin.H{"authenticated": true})
	} else {
		c.JSON(http.StatusUnauthorized, gin.H{"authenticated": false})
	}
}

// generateToken generates a random session token
func generateToken() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

// cleanupSessions removes expired sessions periodically
func cleanupSessions() {
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()

	for range ticker.C {
		sessionsMu.Lock()
		now := time.Now()
		for token, expiry := range activeSessions {
			if now.After(expiry) {
				delete(activeSessions, token)
			}
		}
		sessionsMu.Unlock()
	}
}

// setCookieWithSameSite sets a cookie with proper SameSite attribute for ngrok
func setCookieWithSameSite(c *gin.Context, name, value string, maxAge int, path, domain string, secure, httpOnly bool) {
	// Build cookie string
	cookie := fmt.Sprintf("%s=%s", name, value)
	if maxAge > 0 {
		cookie += fmt.Sprintf("; Max-Age=%d", maxAge)
	} else if maxAge < 0 {
		cookie += "; Max-Age=0"
	}
	if path != "" {
		cookie += fmt.Sprintf("; Path=%s", path)
	}
	if domain != "" {
		cookie += fmt.Sprintf("; Domain=%s", domain)
	}
	if secure {
		cookie += "; Secure"
	}
	if httpOnly {
		cookie += "; HttpOnly"
	}
	// Set SameSite=None for ngrok (cross-origin), SameSite=Lax for localhost
	if secure {
		cookie += "; SameSite=None"
	} else {
		cookie += "; SameSite=Lax"
	}
	c.Header("Set-Cookie", cookie)
}

// getNextOrderID gets the next order ID by finding the highest orderId in the database
func getNextOrderID(ctx context.Context) (int, error) {
	// Find the order with the highest orderId
	findOptions := options.FindOne()
	findOptions.SetSort(bson.D{primitive.E{Key: "orderId", Value: -1}})

	var highestOrder Order
	err := ordersCollection.FindOne(ctx, bson.M{}, findOptions).Decode(&highestOrder)

	if err != nil {
		if err == mongo.ErrNoDocuments {
			// No orders exist, start from 1
			return 1, nil
		}
		return 0, err
	}

	// Return the next order ID
	return highestOrder.OrderID + 1, nil
}
