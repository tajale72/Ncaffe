# Subu Bakery Website

A beautiful, responsive bakery website built with HTML, CSS, JavaScript, and Go (Gin framework).

## Features

- 🍰 Browse bakery products by category
- 🛒 Add products to cart with quantity management
- 📦 Place orders with customer information
- 🎫 View all orders in a beautiful tickets view
- 💾 Orders stored in MongoDB database
- 📱 Fully responsive design for mobile and desktop
- ⚡ Fast and modern UI/UX

## Project Structure

```
Ncaffe/
├── main.go              # Go backend server with Gin
├── go.mod              # Go module dependencies
├── docker-compose.yml  # Docker Compose config for MongoDB
├── templates/          # HTML templates
│   ├── index.html      # Main shop page
│   └── orders.html     # Orders tickets view
├── static/             # Static assets
│   ├── style.css       # Responsive CSS styles
│   ├── script.js       # Frontend JavaScript
│   ├── orders.css      # Orders page styles
│   └── orders.js       # Orders page JavaScript
└── README.md
```

## Prerequisites

- Go 1.21 or higher
- Docker and Docker Compose (for MongoDB) OR MongoDB installed locally
- A web browser

## Installation & Running

### Option 1: Using Docker Compose (Recommended)

1. **Start MongoDB with Docker Compose:**
   ```bash
   docker-compose up -d
   ```
   This will start MongoDB in a container on `localhost:27017`

2. **Install Go dependencies:**
   ```bash
   go mod download
   ```

3. **Run the server:**
   ```bash
   go run main.go
   ```

4. **Open your browser:**
   - Main shop: `http://localhost:8080`
   - Orders view: `http://localhost:8080/orders`

5. **Stop MongoDB (when done):**
   ```bash
   docker-compose down
   ```

### Option 2: Using Local MongoDB

1. **Install and start MongoDB:**
   - Make sure MongoDB is installed and running on `localhost:27017`
   - Or set the `MONGODB_URI` environment variable with your connection string
   - Example: `export MONGODB_URI="mongodb://localhost:27017"`

2. **Install dependencies:**
   ```bash
   go mod download
   ```

3. **Run the server:**
   ```bash
   go run main.go
   ```

4. **Open your browser:**
   - Main shop: `http://localhost:8080`
   - Orders view: `http://localhost:8080/orders`

## API Endpoints

### Products
- `GET /api/products` - Get all products
- `GET /api/products/:id` - Get a specific product

### Orders (Protected - Requires Authentication)
- `POST /api/orders` - Create a new order (public)
- `GET /api/orders` - Get all orders (protected)
- `GET /api/orders/:id` - Get a specific order (protected)
- `POST /api/orders/:id/deliver` - Mark order as delivered (protected)
- `GET /api/delivered` - Get all delivered orders (protected)

### Authentication
- `POST /api/auth/login` - Admin login
- `POST /api/auth/logout` - Admin logout
- `GET /api/auth/check` - Check authentication status

## Features in Detail

### Product Browsing
- Filter products by category (Cookies, Cakes, Pastries, etc.)
- View product details including name, description, and price
- Responsive grid layout that adapts to screen size

### Shopping Cart
- Add/remove items from cart
- Adjust quantities
- View cart total
- Cart persists in browser localStorage

### Order Placement
- Customer information form (name, email, phone, address)
- Order confirmation with order ID
- Orders automatically saved to MongoDB

### Orders View
- Beautiful ticket-style display of all orders
- View customer information, order items, and totals
- Color-coded status indicators (pending, completed, cancelled)
- Auto-refresh every 30 seconds
- Responsive grid layout
- **Admin authentication required** - Login with admin credentials to access
- Mark orders as delivered - Moves orders to delivered collection for tracking

## Technologies Used

- **Backend:** Go 1.21, Gin Web Framework
- **Frontend:** HTML5, CSS3, Vanilla JavaScript
- **Database:** MongoDB (via official Go driver)
- **Storage:** Products in-memory, Orders in MongoDB

## Customization

### Adding Products
Edit the `init()` function in `main.go` to add more products to the initial product list.

### Styling
Modify `static/style.css` to change colors, fonts, and layout. The CSS uses CSS variables for easy theming.

### MongoDB Configuration

**Using Docker Compose:**
- MongoDB runs in a container with persistent volumes
- Data is stored in Docker volumes and persists between container restarts
- Default connection: `mongodb://localhost:27017`

**Using Custom MongoDB:**
- Set the `MONGODB_URI` environment variable
- Example: `export MONGODB_URI="mongodb://user:pass@host:27017/dbname"`

**Docker Compose Commands:**
- Start: `docker-compose up -d`
- Stop: `docker-compose down`
- View logs: `docker-compose logs -f mongodb`
- Remove volumes (clean data): `docker-compose down -v`

### Admin Authentication

**Default Credentials:**
- Username: `admin`
- Password: `bakery123`

**Custom Credentials:**
Set environment variables before starting the server:
```bash
export ADMIN_USERNAME="your_username"
export ADMIN_PASSWORD="your_secure_password"
```

**Features:**
- Orders page requires admin login
- Session tokens valid for 24 hours
- Secure cookie-based authentication
- Logout functionality

### Backend Logic
Extend `main.go` to add features like:
- User authentication
- Payment processing
- Order status updates
- Email notifications
- Product management (add/edit/delete)

## License

This project is open source and available for personal and commercial use.

