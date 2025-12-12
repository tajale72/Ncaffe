package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
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
	ID          primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	ProductID   int                `bson:"productId" json:"productId"`
	Name        string             `bson:"name" json:"name"`
	Description string             `bson:"description" json:"description"`
	Price       float64            `bson:"price" json:"price"`
	Image       string             `bson:"image" json:"image"` // Base64 encoded image or emoji
	Category    string             `bson:"category" json:"category"`
	CreatedAt   time.Time          `bson:"createdAt,omitempty" json:"createdAt,omitempty"`
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
	productsCollection  *mongo.Collection
	ordersCollection    *mongo.Collection
	deliveredCollection *mongo.Collection
	adminUsername       string
	adminPassword       string
	activeSessions      = make(map[string]time.Time)
	sessionsMu          sync.RWMutex
	productIDCounter    = 0
)

// init function removed - products now loaded from MongoDB

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
	productsCollection = client.Database("sububakery").Collection("products")
	ordersCollection = client.Database("sububakery").Collection("orders")
	deliveredCollection = client.Database("sububakery").Collection("delivered")

	// Load products from MongoDB or initialize with defaults
	loadProductsFromDB()

	// Get admin credentials from environment or use defaults
	adminUsername = getEnv("ADMIN_USERNAME", "admin")
	adminPassword = getEnv("ADMIN_PASSWORD", "admin")

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
	// Serve uploaded images
	router.Static("/uploads", "./uploads")
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
			protected.POST("/products", createProduct)
			protected.PUT("/products/:id", updateProduct)
			protected.DELETE("/products/:id", deleteProduct)
		}
	}

	// Start server
	fmt.Println("Subu Bakery server starting on :8080")
	log.Fatal(router.Run(":8085"))
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// getProducts returns all products
func getProducts(c *gin.Context) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cursor, err := productsCollection.Find(ctx, bson.M{})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch products"})
		return
	}
	defer cursor.Close(ctx)

	var productsList []Product
	if err = cursor.All(ctx, &productsList); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to decode products"})
		return
	}

	c.JSON(http.StatusOK, productsList)
}

// getProduct returns a single product by ID
func getProduct(c *gin.Context) {
	id := c.Param("id")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Try to parse as ObjectID first
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		// Try as productId (int)
		var product Product
		err = productsCollection.FindOne(ctx, bson.M{"productId": id}).Decode(&product)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Product not found"})
			return
		}
		c.JSON(http.StatusOK, product)
		return
	}

	var product Product
	err = productsCollection.FindOne(ctx, bson.M{"_id": objectID}).Decode(&product)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			c.JSON(http.StatusNotFound, gin.H{"error": "Product not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch product"})
		return
	}

	c.JSON(http.StatusOK, product)
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
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	for _, item := range orderReq.Items {
		var product Product
		err := productsCollection.FindOne(ctx, bson.M{"productId": item.ProductID}).Decode(&product)
		if err == nil {
			total += product.Price * float64(item.Quantity)
		}
	}

	// Get next order ID from database
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

// loadProductsFromDB loads products from MongoDB or initializes with defaults
func loadProductsFromDB() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	cursor, err := productsCollection.Find(ctx, bson.M{})
	if err != nil {
		log.Println("Error loading products from DB, using defaults:", err)
		initializeDefaultProducts(ctx)
		return
	}
	defer cursor.Close(ctx)

	var productsList []Product
	if err = cursor.All(ctx, &productsList); err != nil {
		log.Println("Error decoding products, using defaults:", err)
		initializeDefaultProducts(ctx)
		return
	}

	if len(productsList) == 0 {
		initializeDefaultProducts(ctx)
		return
	}

	productsMu.Lock()
	products = productsList
	// Set productIDCounter to highest productId
	for _, p := range productsList {
		if p.ProductID > productIDCounter {
			productIDCounter = p.ProductID
		}
	}
	productsMu.Unlock()
}

// initializeDefaultProducts creates default products if database is empty
func initializeDefaultProducts(ctx context.Context) {
	defaultProducts := []Product{
		{ProductID: 1, Name: "Chocolate Chip Cookies", Description: "Freshly baked cookies with premium chocolate chips", Price: 8.99, Image: "🍪", Category: "Cookies", CreatedAt: time.Now()},
		{ProductID: 2, Name: "Blueberry Muffins", Description: "Moist muffins bursting with fresh blueberries", Price: 6.99, Image: "🧁", Category: "Muffins", CreatedAt: time.Now()},
		{ProductID: 3, Name: "Croissant", Description: "Buttery, flaky French croissant", Price: 4.99, Image: "🥐", Category: "Pastries", CreatedAt: time.Now()},
		{ProductID: 4, Name: "Chocolate Cake", Description: "Rich chocolate layer cake with buttercream frosting", Price: 24.99, Image: "🎂", Category: "Cakes", CreatedAt: time.Now()},
		{ProductID: 5, Name: "Apple Pie", Description: "Homemade apple pie with cinnamon", Price: 18.99, Image: "🥧", Category: "Pies", CreatedAt: time.Now()},
		{ProductID: 6, Name: "Bagels", Description: "Fresh New York style bagels (pack of 6)", Price: 7.99, Image: "🥯", Category: "Breads", CreatedAt: time.Now()},
		{ProductID: 7, Name: "Cinnamon Roll", Description: "Warm cinnamon rolls with cream cheese glaze", Price: 5.99, Image: "🍩", Category: "Pastries", CreatedAt: time.Now()},
		{ProductID: 8, Name: "Strawberry Tart", Description: "Delicate tart with fresh strawberries", Price: 12.99, Image: "🍓", Category: "Tarts", CreatedAt: time.Now()},
	}

	var docs []interface{}
	for _, p := range defaultProducts {
		p.ID = primitive.NewObjectID()
		docs = append(docs, p)
	}

	_, err := productsCollection.InsertMany(ctx, docs)
	if err != nil {
		log.Println("Error inserting default products:", err)
	}

	productsMu.Lock()
	products = defaultProducts
	productIDCounter = 8
	productsMu.Unlock()
}

// createProduct creates a new product (multipart/form-data, file upload)
func createProduct(c *gin.Context) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Read text fields from form-data
	name := c.PostForm("name")
	description := c.PostForm("description")
	priceStr := c.PostForm("price")
	category := c.PostForm("category")

	// Convert price string → float64
	price, _ := strconv.ParseFloat(priceStr, 64)

	// Validate required fields
	if name == "" || category == "" || price <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Name, category, and valid price are required"})
		return
	}

	// Handle image upload
	file, err := c.FormFile("image")
	var imageURL string
	host := c.Request.Host
	scheme := "http"
	if c.Request.TLS != nil {
		scheme = "https"
	}

	fullDomain := fmt.Sprintf("%s://%s", scheme, host)

	if err == nil && file != nil {
		// Save image to /uploads folder
		filename := fmt.Sprintf("uploads/%d_%s", time.Now().Unix(), file.Filename)
		if err := c.SaveUploadedFile(file, filename); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Image upload failed"})
			return
		}
		imageURL = fullDomain + "/" + filename
	} else {
		// no image uploaded → default placeholder
		imageURL = "/images/default.png"
	}

	// Generate productID
	nextProductID, err := getNextProductID(ctx)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate product ID"})
		return
	}

	// Create product struct
	product := Product{
		ID:          primitive.NewObjectID(),
		ProductID:   nextProductID,
		Name:        name,
		Description: description,
		Price:       price,
		Image:       imageURL,
		Category:    category,
		CreatedAt:   time.Now(),
	}

	// Save to MongoDB
	_, err = productsCollection.InsertOne(ctx, product)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save product"})
		return
	}

	// Update cache
	productsMu.Lock()
	products = append(products, product)
	productsMu.Unlock()

	c.JSON(http.StatusCreated, product)
}

// getNextProductID gets the next product ID
func getNextProductID(ctx context.Context) (int, error) {
	findOptions := options.FindOne()
	findOptions.SetSort(bson.D{primitive.E{Key: "productId", Value: -1}})

	var highestProduct Product
	err := productsCollection.FindOne(ctx, bson.M{}, findOptions).Decode(&highestProduct)

	if err != nil {
		if err == mongo.ErrNoDocuments {
			return 1, nil
		}
		return 0, err
	}

	return highestProduct.ProductID + 1, nil
}

// updateProduct updates an existing product
func updateProduct(c *gin.Context) {
	id := c.Param("id")
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid product ID format"})
		return
	}

	// Read text fields from multipart/form-data
	name := c.PostForm("name")
	description := c.PostForm("description")
	priceStr := c.PostForm("price")
	category := c.PostForm("category")

	// Convert price from string → float
	price, _ := strconv.ParseFloat(priceStr, 64)

	// Handle optional image
	file, err := c.FormFile("image")

	var imageURL string

	if err == nil && file != nil {
		// Save the new image
		filename := "uploads/" + file.Filename
		if err := c.SaveUploadedFile(file, filename); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Image upload failed"})
			return
		}
		imageURL = "/" + filename
	}

	// Build update object
	update := bson.M{
		"$set": bson.M{
			"name":        name,
			"description": description,
			"price":       price,
			"category":    category,
		},
	}

	// Only update the image if new file was uploaded
	if imageURL != "" {
		update["$set"].(bson.M)["image"] = imageURL
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	result := productsCollection.FindOneAndUpdate(ctx, bson.M{"_id": objectID}, update)
	if result.Err() != nil {
		if result.Err() == mongo.ErrNoDocuments {
			c.JSON(http.StatusNotFound, gin.H{"error": "Product not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update product"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Product updated successfully"})
}

// deleteProduct deletes a product
func deleteProduct(c *gin.Context) {
	id := c.Param("id")
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid product ID format"})
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err = productsCollection.DeleteOne(ctx, bson.M{"_id": objectID})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete product"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Product deleted successfully"})
}
