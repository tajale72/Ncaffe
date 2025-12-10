// Global state
let products = [];
let cart = [];
let filteredProducts = [];

// Initialize app
document.addEventListener('DOMContentLoaded', async () => {
    setupEventListeners();
    await loadProducts();
    loadCartFromStorage();
    updateCartDisplay();
});

// Helper function to create fetch options with ngrok header
function getFetchOptions(method = 'GET', body = null) {
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
    
    return options;
}

// Load products from API
async function loadProducts() {
    try {
        const response = await fetch('/api/products', getFetchOptions());
        products = await response.json();
        filteredProducts = products;
        displayProducts(products);
    } catch (error) {
        console.error('Error loading products:', error);
        showError('Failed to load products. Please refresh the page.');
    }
}

// Display products in grid
function displayProducts(productsToShow) {
    const grid = document.getElementById('products-grid');
    
    if (productsToShow.length === 0) {
        grid.innerHTML = '<p style="grid-column: 1/-1; text-align: center; padding: 2rem;">No products found in this category.</p>';
        return;
    }

    grid.innerHTML = productsToShow.map(product => `
        <div class="product-card">
            <div class="product-image">${product.image}</div>
            <div class="product-name">${product.name}</div>
            <div class="product-description">${product.description}</div>
            <div class="product-footer">
                <div class="product-price">$${product.price.toFixed(2)}</div>
                <button class="add-to-cart-btn" onclick="addToCart(${product.id})">
                    Add to Cart
                </button>
            </div>
        </div>
    `).join('');
}

// Filter products by category
function filterProducts(category) {
    if (category === 'all') {
        filteredProducts = products;
    } else {
        filteredProducts = products.filter(p => p.category === category);
    }
    displayProducts(filteredProducts);
}

// Add product to cart
function addToCart(productId) {
    const product = products.find(p => p.id === productId);
    if (!product) return;

    const existingItem = cart.find(item => item.productId === productId);
    
    if (existingItem) {
        existingItem.quantity++;
    } else {
        cart.push({
            productId: productId,
            quantity: 1,
            product: product
        });
    }

    saveCartToStorage();
    updateCartDisplay();
    showNotification(`${product.name} added to cart!`);
}

// Remove item from cart
function removeFromCart(productId) {
    cart = cart.filter(item => item.productId !== productId);
    saveCartToStorage();
    updateCartDisplay();
}

// Update quantity
function updateQuantity(productId, change) {
    const item = cart.find(item => item.productId === productId);
    if (!item) return;

    item.quantity += change;
    
    if (item.quantity <= 0) {
        removeFromCart(productId);
    } else {
        saveCartToStorage();
        updateCartDisplay();
    }
}

// Update cart display
function updateCartDisplay() {
    const cartItems = document.getElementById('cart-items');
    const cartTotal = document.getElementById('cart-total');
    const cartCount = document.getElementById('cart-count');
    
    // Filter out items without product data
    const validCart = cart.filter(item => item.product && item.product.name);
    
    cartCount.textContent = validCart.reduce((sum, item) => sum + item.quantity, 0);

    if (validCart.length === 0) {
        cartItems.innerHTML = '<p class="empty-cart">Your cart is empty</p>';
        cartTotal.style.display = 'none';
        // Update cart to remove invalid items
        if (cart.length !== validCart.length) {
            cart = validCart;
            saveCartToStorage();
        }
        return;
    }

    cartItems.innerHTML = validCart.map(item => `
        <div class="cart-item">
            <div class="cart-item-info">
                <div class="cart-item-name">${item.product.name}</div>
                <div class="cart-item-price">$${item.product.price.toFixed(2)} each</div>
            </div>
            <div class="cart-item-controls">
                <div class="quantity-control">
                    <button class="quantity-btn" onclick="updateQuantity(${item.productId}, -1)">-</button>
                    <span class="quantity">${item.quantity}</span>
                    <button class="quantity-btn" onclick="updateQuantity(${item.productId}, 1)">+</button>
                </div>
                <button class="remove-btn" onclick="removeFromCart(${item.productId})">Remove</button>
            </div>
        </div>
    `).join('');

    const total = validCart.reduce((sum, item) => sum + (item.product.price * item.quantity), 0);
    document.getElementById('total-amount').textContent = total.toFixed(2);
    cartTotal.style.display = 'block';
    
    // Update cart to remove invalid items if any were filtered out
    if (cart.length !== validCart.length) {
        cart = validCart;
        saveCartToStorage();
    }
}

// Setup event listeners
function setupEventListeners() {
    // Filter buttons
    document.querySelectorAll('.filter-btn').forEach(btn => {
        btn.addEventListener('click', () => {
            document.querySelectorAll('.filter-btn').forEach(b => b.classList.remove('active'));
            btn.classList.add('active');
            filterProducts(btn.dataset.category);
        });
    });

    // Checkout button
    document.getElementById('checkout-btn').addEventListener('click', () => {
        document.getElementById('cart').scrollIntoView({ behavior: 'smooth' });
        setTimeout(() => {
            document.getElementById('checkout').style.display = 'block';
            document.getElementById('checkout').scrollIntoView({ behavior: 'smooth' });
        }, 300);
    });

    // Cancel checkout
    document.getElementById('cancel-checkout').addEventListener('click', () => {
        document.getElementById('checkout').style.display = 'none';
    });

    // Checkout form
    document.getElementById('checkout-form').addEventListener('submit', handleCheckout);

    // New order button
    document.getElementById('new-order-btn').addEventListener('click', () => {
        cart = [];
        saveCartToStorage();
        updateCartDisplay();
        document.getElementById('order-success').style.display = 'none';
        document.getElementById('checkout-form').reset();
        window.scrollTo({ top: 0, behavior: 'smooth' });
    });
}

// Handle checkout
async function handleCheckout(e) {
    e.preventDefault();

    if (cart.length === 0) {
        showError('Your cart is empty!');
        return;
    }

    const formData = {
        customer: {
            name: document.getElementById('name').value,
            email: document.getElementById('email').value,
            phone: document.getElementById('phone').value,
            address: document.getElementById('address').value
        },
        items: cart.map(item => ({
            productId: item.productId,
            quantity: item.quantity
        }))
    };

    try {
        const response = await fetch('/api/orders', getFetchOptions('POST', formData));

        if (!response.ok) {
            throw new Error('Failed to place order');
        }

        const order = await response.json();
        
        // Show success message
        const orderIdDisplay = order.orderId || order.id || 'N/A';
        document.getElementById('order-id').textContent = orderIdDisplay;
        document.getElementById('checkout').style.display = 'none';
        document.getElementById('order-success').style.display = 'block';
        document.getElementById('order-success').scrollIntoView({ behavior: 'smooth' });

        // Clear cart
        cart = [];
        saveCartToStorage();
        updateCartDisplay();

    } catch (error) {
        console.error('Error placing order:', error);
        showError('Failed to place order. Please try again.');
    }
}

// Save cart to localStorage
function saveCartToStorage() {
    localStorage.setItem('subuBakeryCart', JSON.stringify(cart));
}

// Load cart from localStorage
function loadCartFromStorage() {
    const saved = localStorage.getItem('subuBakeryCart');
    if (saved) {
        cart = JSON.parse(saved);
        // Re-attach product objects (only if products are loaded)
        if (products.length > 0) {
            cart.forEach(item => {
                item.product = products.find(p => p.id === item.productId);
            });
            // Remove any items where product wasn't found (product might have been removed)
            cart = cart.filter(item => item.product !== undefined);
        }
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
        border-radius: 5px;
        box-shadow: 0 4px 10px rgba(0,0,0,0.2);
        z-index: 1000;
        animation: slideIn 0.3s ease;
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
        border-radius: 5px;
        box-shadow: 0 4px 10px rgba(0,0,0,0.2);
        z-index: 1000;
        animation: slideIn 0.3s ease;
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

