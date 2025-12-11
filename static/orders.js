// Global state
let orders = [];
let products = [];
let authToken = null;

// Helper function to create fetch options with ngrok header
function getFetchOptions(method = 'GET', body = null, includeAuth = false) {
    const options = {
        method: method,
        headers: {
            'ngrok-skip-browser-warning': '1'
        }
    };

    if (body) {
        options.headers['Content-Type'] = 'application/json';
        options.body = JSON.stringify(body);
    }

    if (includeAuth) {
        const token = getAuthToken();
        if (token) {
            options.headers['Authorization'] = `Bearer ${token}`;
        }
    }

    return options;
}

// Initialize
document.addEventListener('DOMContentLoaded', async () => {
    // Check authentication first
    await checkAuthentication();
    loadProducts();
    setupEventListeners();
});

document.getElementById("close-login").addEventListener("click", () => {
    window.location.href = "/"; // Go to home page
});


// Load products (needed to display product names in orders)
async function loadProducts() {
    try {
        const response = await fetch('/api/products', getFetchOptions());
        products = await response.json();
    } catch (error) {
        console.error('Error loading products:', error);
    }
}

// Check authentication status
async function checkAuthentication() {
    try {
        const response = await fetch('/api/auth/check', getFetchOptions());
        const data = await response.json();

        if (data.authenticated) {
            // Get token from cookie or localStorage
            authToken = getAuthToken();
            showOrdersPage();
        } else {
            showLoginModal();
        }
    } catch (error) {
        console.error('Error checking auth:', error);
        showLoginModal();
    }
}

// Get auth token from cookie or localStorage
function getAuthToken() {
    // Try to get from cookie
    const cookies = document.cookie.split(';');
    for (let cookie of cookies) {
        const [name, value] = cookie.trim().split('=');
        if (name === 'auth_token') {
            return value;
        }
    }
    // Try localStorage as fallback
    return localStorage.getItem('auth_token');
}

// Show login modal
function showLoginModal() {
    document.getElementById('login-modal').style.display = 'flex';
    document.getElementById('orders-section').style.display = 'none';
}

// Show orders page
function showOrdersPage() {
    document.getElementById('login-modal').style.display = 'none';
    document.getElementById('orders-section').style.display = 'block';
    loadOrders();
}

// Handle login
async function handleLogin(e) {
    e.preventDefault();

    const username = document.getElementById('username').value;
    const password = document.getElementById('password').value;
    const errorDiv = document.getElementById('login-error');

    try {
        const response = await fetch('/api/auth/login', getFetchOptions('POST', { username, password }));

        const data = await response.json();

        if (!response.ok) {
            errorDiv.textContent = data.error || 'Login failed';
            errorDiv.style.display = 'block';
            return;
        }

        // Store token
        authToken = data.token;
        localStorage.setItem('auth_token', data.token);

        // Hide login modal and show orders
        showOrdersPage();
    } catch (error) {
        console.error('Login error:', error);
        errorDiv.textContent = 'Failed to login. Please try again.';
        errorDiv.style.display = 'block';
    }
}

// Handle logout
async function handleLogout() {
    try {
        await fetch('/api/auth/logout', getFetchOptions('POST', null, true));
    } catch (error) {
        console.error('Logout error:', error);
    }

    // Clear token
    authToken = null;
    localStorage.removeItem('auth_token');

    // Show login modal
    showLoginModal();
    document.getElementById('login-form').reset();
}

// Load orders from API
async function loadOrders() {
    const container = document.getElementById('orders-container');
    container.innerHTML = '<div class="loading">Loading orders...</div>';

    try {
        const response = await fetch('/api/orders', getFetchOptions('GET', null, true));

        if (response.status === 401) {
            // Not authenticated, show login
            showLoginModal();
            return;
        }

        if (!response.ok) {
            throw new Error('Failed to load orders');
        }
        orders = await response.json();
        displayOrders(orders);
    } catch (error) {
        console.error('Error loading orders:', error);
        if (error.message.includes('401') || error.message.includes('Unauthorized')) {
            showLoginModal();
        } else {
            container.innerHTML = `
                <div class="empty-orders">
                    <h3>Error Loading Orders</h3>
                    <p>${error.message}</p>
                </div>
            `;
        }
    }
}

// Display orders in tickets format
function displayOrders(ordersToShow) {
    const container = document.getElementById('orders-container');

    if (ordersToShow.length === 0) {
        container.innerHTML = `
            <div class="empty-orders">
                <h3>No Orders Yet</h3>
                <p>Orders will appear here once customers start placing them.</p>
            </div>
        `;
        return;
    }

    container.innerHTML = ordersToShow.map(order => createOrderTicket(order)).join('');
}

// Create order ticket HTML
function createOrderTicket(order) {
    const date = new Date(order.createdAt);
    const formattedDate = date.toLocaleString('en-US', {
        year: 'numeric',
        month: 'short',
        day: 'numeric',
        hour: '2-digit',
        minute: '2-digit'
    });

    const statusClass = order.status.toLowerCase();
    // Order ID for display (the numeric orderId)
    const orderId = order.orderId || 'N/A';
    // MongoDB ObjectID (stored in id field as hex string)
    const orderMongoId = order.id || (order._id ? String(order._id) : null);

    // Get product details for items with images and prices
    // Get product details for items with images and prices
    const itemsHtml = order.items.map(item => {
        const product = products.find(p => p.productId === item.productId);
        const productName = product ? product.name : `Product #${item.productId}`;
        const productImage = product ? product.image : '📦';
        const productPrice = product ? product.price : 0;
        const itemTotal = productPrice * item.quantity;

        // Determine image display format
        let imageDisplay = '';

        if (!productImage) {
            imageDisplay = `<div class="order-item-image">📦</div>`;
        }
        else if (productImage.startsWith('data:image')) {
            imageDisplay = `<img src="${productImage}" class="order-item-image-img" alt="${productName}">`;
        }
        else if (productImage.startsWith('http://') || productImage.startsWith('https://')) {
            imageDisplay = `<img src="${productImage}" class="order-item-image-img" alt="${productName}">`;
        }
        else if (productImage.length <= 4) {
            imageDisplay = `<div class="order-item-image">${productImage}</div>`;
        }
        else {
            imageDisplay = `<div class="order-item-image">📦</div>`;
        }

        return `
        <li class="order-item-row">
            ${imageDisplay}
            <div class="order-item-details">
                <div class="order-item-name">${productName}</div>
                <div class="order-item-price-info">
                    <span class="order-item-unit-price">$${productPrice.toFixed(2)} each</span>
                    <span class="order-item-quantity">× ${item.quantity}</span>
                </div>
            </div>
            <div class="order-item-total">$${itemTotal.toFixed(2)}</div>
        </li>
    `;
    }).join('');


    // Only show delivered button if order is not already delivered and we have a valid MongoDB ID
    const deliveredButton = (statusClass !== 'delivered' && orderMongoId) ? `
        <button class="delivered-btn" onclick="markAsDelivered('${orderMongoId}', ${orderId})">
            ✓ Mark as Delivered
        </button>
    ` : '';

    return `
        <div class="order-ticket ${statusClass}">
            <div class="order-ticket-header">
                <div class="order-header-info">
                    <div class="order-id-large">Order #${orderId}</div>
                    <div class="order-date">${formattedDate}</div>
                </div>
                <div class="order-status ${statusClass}">${order.status}</div>
            </div>
            
            <div class="order-summary">
                <div class="order-summary-item">
                    <span class="summary-label">Items:</span>
                    <span class="summary-value">${order.items.length}</span>
                </div>
                <div class="order-summary-item">
                    <span class="summary-label">Total Amount:</span>
                    <span class="summary-value total-highlight">$${order.total.toFixed(2)}</span>
                </div>
            </div>
            
            <div class="order-customer">
                <h4>Customer Information</h4>
                <div class="customer-info">
                    <strong>Name:</strong>
                    <span>${order.customer.name}</span>
                    <strong>Email:</strong>
                    <span>${order.customer.email}</span>
                    <strong>Phone:</strong>
                    <span>${order.customer.phone}</span>
                    <strong>Address:</strong>
                    <span>${order.customer.address}</span>
                </div>
            </div>

            <div class="order-items">
                <h4>Order Items</h4>
                <ul class="order-item-list">
                    ${itemsHtml}
                </ul>
            </div>

            <div class="order-total">
                <span class="order-total-label">Total:</span>
                <span class="order-total-amount">$${order.total.toFixed(2)}</span>
            </div>

            ${deliveredButton ? `<div class="order-actions">${deliveredButton}</div>` : ''}
        </div>
    `;
}

// Mark order as delivered
async function markAsDelivered(orderMongoId, orderId) {
    if (!confirm(`Are you sure you want to mark Order #${orderId} as delivered?`)) {
        return;
    }

    try {
        const response = await fetch(`/api/orders/${orderMongoId}/deliver`, getFetchOptions('POST', null, true));

        if (!response.ok) {
            const error = await response.json();
            throw new Error(error.error || 'Failed to mark order as delivered');
        }

        const result = await response.json();
        showNotification(`Order #${orderId} has been marked as delivered!`);

        // Reload orders to remove the delivered order
        loadOrders();
    } catch (error) {
        console.error('Error marking order as delivered:', error);
        showError(error.message || 'Failed to mark order as delivered. Please try again.');
    }
}

// Show notification
function showNotification(message) {
    const notification = document.createElement('div');
    notification.style.cssText = `
        position: fixed;
        top: 100px;
        right: 20px;
        background: #28a745;
        color: white;
        padding: 1rem 2rem;
        border-radius: 8px;
        box-shadow: 0 4px 15px rgba(0,0,0,0.2);
        z-index: 1000;
        animation: slideIn 0.3s ease;
        font-family: 'Lato', sans-serif;
    `;
    notification.textContent = message;
    document.body.appendChild(notification);

    setTimeout(() => {
        notification.style.animation = 'slideOut 0.3s ease';
        setTimeout(() => notification.remove(), 300);
    }, 3000);
}

// Show error
function showError(message) {
    const error = document.createElement('div');
    error.style.cssText = `
        position: fixed;
        top: 100px;
        right: 20px;
        background: #dc3545;
        color: white;
        padding: 1rem 2rem;
        border-radius: 8px;
        box-shadow: 0 4px 15px rgba(0,0,0,0.2);
        z-index: 1000;
        animation: slideIn 0.3s ease;
        font-family: 'Lato', sans-serif;
    `;
    error.textContent = message;
    document.body.appendChild(error);

    setTimeout(() => {
        error.style.animation = 'slideOut 0.3s ease';
        setTimeout(() => error.remove(), 300);
    }, 5000);
}

// Add CSS animations
const style = document.createElement('style');
style.textContent = `
    @keyframes slideIn {
        from {
            transform: translateX(400px);
            opacity: 0;
        }
        to {
            transform: translateX(0);
            opacity: 1;
        }
    }
    @keyframes slideOut {
        from {
            transform: translateX(0);
            opacity: 1;
        }
        to {
            transform: translateX(400px);
            opacity: 0;
        }
    }
`;
document.head.appendChild(style);

// Setup event listeners
function setupEventListeners() {
    document.getElementById('login-form').addEventListener('submit', handleLogin);
    document.getElementById('logout-btn').addEventListener('click', handleLogout);
    document.getElementById('refresh-btn').addEventListener('click', () => {
        loadOrders();
    });
}

// Auto-refresh every 30 seconds
setInterval(() => {
    loadOrders();
}, 30000);

