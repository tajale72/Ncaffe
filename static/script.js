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
        const token = localStorage.getItem('auth_token');
        if (token) {
            options.headers['Authorization'] = `Bearer ${token}`;
        }
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

    grid.innerHTML = productsToShow.map(product => {
        // Check if image is base64 or emoji
        const imageDisplay = product.image && product.image.startsWith('data:image') 
            ? `<img src="${product.image}" alt="${product.name}" class="product-image-img">` 
            : `<div class="product-image">${product.image || '📦'}</div>`;
        
        return `
        <div class="product-card">
            ${imageDisplay}
            <div class="product-name">${product.name}</div>
            <div class="product-description">${product.description}</div>
            <div class="product-footer">
                <div class="product-price">$${product.price.toFixed(2)}</div>
                <button class="add-to-cart-btn" onclick="addToCart(${product.productId})">
                    Add to Cart
                </button>
            </div>
        </div>
    `;
    }).join('');
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
    const product = products.find(p => p.productId === productId);
    if (!product) return;

    const existingItem = cart.find(item => item.productId === productId);
    
    if (existingItem) {
        existingItem.quantity++;
    } else {
        cart.push({
            productId: product.productId,
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

    // Add product form
    document.getElementById('add-product-form').addEventListener('submit', handleAddProduct);
    document.getElementById('cancel-product-btn').addEventListener('click', () => {
        document.getElementById('add-product').style.display = 'none';
        document.getElementById('add-product-form').reset();
        document.getElementById('image-preview').innerHTML = '<div class="image-preview-empty">No image selected. Click buttons above to add an image.</div>';
        document.getElementById('product-error').style.display = 'none';
    });
    document.getElementById('camera-btn').addEventListener('click', () => {
        const input = document.getElementById('product-image');
        input.setAttribute('capture', 'environment');
        input.click();
    });
    document.getElementById('file-btn').addEventListener('click', () => {
        const input = document.getElementById('product-image');
        input.removeAttribute('capture');
        input.click();
    });
    document.getElementById('product-image').addEventListener('change', handleImagePreview);
    
    // Check if user is admin (has auth token)
    checkAdminAccess();
}

// Check if user has admin access
function checkAdminAccess() {
    const token = localStorage.getItem('auth_token');
    if (token) {
        document.getElementById('add-product-link').style.display = 'block';
        document.getElementById('add-product-link').addEventListener('click', (e) => {
            e.preventDefault();
            document.getElementById('add-product').style.display = 'block';
            document.getElementById('add-product').scrollIntoView({ behavior: 'smooth' });
        });
    }
}

// Handle image preview
function handleImagePreview(e) {
    const file = e.target.files[0];
    const preview = document.getElementById('image-preview');
    
    if (file) {
        // Validate file size (max 5MB)
        if (file.size > 5 * 1024 * 1024) {
            preview.innerHTML = '<div class="error-message">Image is too large. Please choose an image smaller than 5MB.</div>';
            e.target.value = '';
            return;
        }
        
        // Validate file type
        if (!file.type.startsWith('image/')) {
            preview.innerHTML = '<div class="error-message">Please select a valid image file.</div>';
            e.target.value = '';
            return;
        }
        
        const reader = new FileReader();
        reader.onload = (event) => {
            preview.innerHTML = `
                <img src="${event.target.result}" alt="Preview">
                <p style="margin-top: 0.75rem; color: var(--text-medium); font-size: 0.9rem;">
                    ${file.name} (${(file.size / 1024).toFixed(1)} KB)
                </p>
            `;
        };
        reader.onerror = () => {
            preview.innerHTML = '<div class="error-message">Error loading image. Please try again.</div>';
        };
        reader.readAsDataURL(file);
    } else {
        preview.innerHTML = '<div class="image-preview-empty">No image selected. Click buttons above to add an image.</div>';
    }
}

// Handle add product
async function handleAddProduct(e) {
    e.preventDefault();
    
    const formData = {
        name: document.getElementById('product-name').value,
        description: document.getElementById('product-description').value,
        price: parseFloat(document.getElementById('product-price').value),
        category: document.getElementById('product-category').value,
        image: ''
    };
    
    // Get image as base64
    const imageInput = document.getElementById('product-image');
    if (imageInput.files && imageInput.files[0]) {
        const file = imageInput.files[0];
        formData.image = await fileToBase64(file);
    } else {
        // Use default emoji if no image
        formData.image = '📦';
    }
    
    const errorDiv = document.getElementById('product-error');
    
    try {
        const response = await fetch('/api/products', getFetchOptions('POST', formData, true));
        
        if (response.status === 401) {
            errorDiv.textContent = 'Authentication required. Please login first.';
            errorDiv.style.display = 'block';
            return;
        }
        
        const data = await response.json();
        
        if (!response.ok) {
            errorDiv.textContent = data.error || 'Failed to add product';
            errorDiv.style.display = 'block';
            return;
        }
        
        // Success - reload products and reset form
        showNotification('Product added successfully!');
        await loadProducts();
        document.getElementById('add-product-form').reset();
        document.getElementById('image-preview').innerHTML = '<div class="image-preview-empty">No image selected. Click buttons above to add an image.</div>';
        document.getElementById('add-product').style.display = 'none';
        document.getElementById('products').scrollIntoView({ behavior: 'smooth' });
    } catch (error) {
        console.error('Error adding product:', error);
        errorDiv.textContent = 'Failed to add product. Please try again.';
        errorDiv.style.display = 'block';
    }
}

// Convert file to base64
function fileToBase64(file) {
    return new Promise((resolve, reject) => {
        const reader = new FileReader();
        reader.onload = () => resolve(reader.result);
        reader.onerror = reject;
        reader.readAsDataURL(file);
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
        const response = await fetch('/api/orders', getFetchOptions('POST', formData, false));

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
            item.product = products.find(p => p.productId === item.productId);
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

