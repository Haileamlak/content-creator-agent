/**
 * Conca Main Application Logic
 */

const API_BASE = '/api';

const state = {
    user: null,
    token: localStorage.getItem('conca_token'),
    brands: [],
    currentView: 'overview'
};

// --- Initialization ---

document.addEventListener('DOMContentLoaded', () => {
    initAuth();
    initNavigation();

    if (state.token) {
        showView('dashboard-view');
        loadDashboard();
    } else {
        showView('auth-view');
    }
});

// --- Auth Functions ---

function initAuth() {
    const authForm = document.getElementById('auth-form');
    const toggleContainer = document.querySelector('.toggle-auth');
    let isRegisterMode = false;

    toggleContainer.addEventListener('click', (e) => {
        if (e.target.tagName !== 'A') return;
        e.preventDefault();
        isRegisterMode = !isRegisterMode;

        const h1 = document.querySelector('#auth-view h1');
        const btn = authForm.querySelector('.btn-primary');

        if (isRegisterMode) {
            h1.textContent = 'Create Account';
            btn.textContent = 'Sign Up';
            toggleContainer.innerHTML = 'Already have an account? <a href="#" id="show-login">Login</a>';
        } else {
            h1.textContent = 'Welcome to Conca';
            btn.textContent = 'Sign In';
            toggleContainer.innerHTML = 'Don\'t have an account? <a href="#" id="show-register">Register</a>';
        }
    });

    authForm.addEventListener('submit', async (e) => {
        e.preventDefault();
        const email = document.getElementById('email').value;
        const password = document.getElementById('password').value;

        const endpoint = isRegisterMode ? '/auth/register' : '/auth/login';

        try {
            const data = await request(endpoint, 'POST', { email, password });
            state.token = data.token;
            localStorage.setItem('conca_token', data.token);
            showNotification(isRegisterMode ? 'Account created!' : 'Welcome back!', 'success');
            showView('dashboard-view');
            loadDashboard();
        } catch (err) {
            showNotification(err.message, 'error');
        }
    });

    document.getElementById('logout-btn').addEventListener('click', () => {
        state.token = null;
        localStorage.removeItem('conca_token');
        showView('auth-view');
    });
}

// --- Navigation ---

function initNavigation() {
    const navLinks = document.querySelectorAll('.nav-links li');
    navLinks.forEach(link => {
        link.addEventListener('click', () => {
            navLinks.forEach(l => l.classList.remove('active'));
            link.classList.add('active');
            const view = link.dataset.view;
            state.currentView = view;
            renderView(view);
        });
    });
}

function showView(viewId) {
    document.querySelectorAll('.view').forEach(v => v.classList.add('hidden'));
    document.getElementById(viewId).classList.remove('hidden');
}

// --- API Helpers ---

async function request(endpoint, method = 'GET', body = null) {
    const headers = { 'Content-Type': 'application/json' };
    if (state.token) headers['Authorization'] = `Bearer ${state.token}`;

    const config = { method, headers };
    if (body) config.body = JSON.stringify(body);

    const res = await fetch(`${API_BASE}${endpoint}`, config);
    const data = await res.json();

    if (!res.ok) throw new Error(data.error || 'Request failed');
    return data;
}

// --- Rendering Logic ---

async function loadDashboard() {
    renderView('overview');
}

async function renderView(view) {
    const container = document.getElementById('view-container');
    const title = document.getElementById('page-title');
    container.innerHTML = '';

    switch (view) {
        case 'overview':
            title.textContent = 'Dashboard Overview';
            await renderOverview(container);
            break;
        case 'brands':
            title.textContent = 'Brand Management';
            await renderBrands(container);
            break;
        case 'calendar':
            title.textContent = 'Content Calendar';
            container.innerHTML = '<div class="glass card">Calendar coming soon in Phase 3.</div>';
            break;
        case 'analytics':
            title.textContent = 'Performance Analytics';
            container.innerHTML = '<div class="glass card">Analytics dashboard coming soon.</div>';
            break;
    }
}

async function renderOverview(container) {
    try {
        const brands = await request('/brands');
        state.brands = brands;

        container.innerHTML = `
            <div class="glass card stat-item animate-fade">
                <span class="stat-label">Active Brands</span>
                <span class="stat-value">${brands.length}</span>
            </div>
            <div class="glass card stat-item animate-fade" style="animation-delay: 0.1s">
                <span class="stat-label">Total Posts</span>
                <span class="stat-value">...</span>
            </div>
            <div class="glass card stat-item animate-fade" style="animation-delay: 0.2s">
                <span class="stat-label">Avg. Engagement</span>
                <span class="stat-value">N/A</span>
            </div>
        `;
    } catch (err) {
        showNotification('Failed to load dashboard data', 'error');
    }
}

async function renderBrands(container) {
    container.innerHTML = `
        <div class="glass card brand-card" style="display: flex; justify-content: center; align-items: center; border: 2px dashed var(--glass-border); cursor: pointer;" id="new-brand-btn">
            <div style="text-align: center">
                <span style="font-size: 3rem; color: var(--primary-light)">+</span>
                <p style="font-weight: 600">Create New Brand</p>
            </div>
        </div>
    `;

    container.className = 'brand-grid';
    try {
        const brands = await request('/brands');

        brands.forEach((brand, idx) => {
            const card = document.createElement('div');
            card.className = 'glass card brand-card animate-fade';
            card.style.animationDelay = `${idx * 0.1}s`;
            card.innerHTML = `
                <h3>${brand.name}</h3>
                <div class="brand-meta">
                    <p><strong>Industry:</strong> ${brand.industry}</p>
                    <p><strong>Voice:</strong> ${brand.voice}</p>
                    <p><strong>Interval:</strong> ${brand.schedule_interval_hours || 4}h</p>
                </div>
                <div class="brand-actions">
                    <button class="btn-small btn-run" data-id="${brand.id}" style="background: var(--primary)">Run Cycle</button>
                    <button class="btn-small btn-sync" data-id="${brand.id}" style="background: var(--secondary)">Sync</button>
                    <button class="btn-small btn-posts" data-id="${brand.id}" style="background: rgba(255,255,255,0.1)">Posts</button>
                </div>
            `;
            container.appendChild(card);
        });

        // Event Listeners
        document.getElementById('new-brand-btn').addEventListener('click', () => renderBrandWizard(container));

        container.querySelectorAll('.btn-run').forEach(btn => {
            btn.addEventListener('click', async () => {
                const id = btn.dataset.id;
                try {
                    btn.disabled = true;
                    btn.textContent = 'Queuing...';
                    await request(`/brands/${id}/run`, 'POST');
                    showNotification(`Job queued for ${id}`, 'success');
                } catch (err) {
                    showNotification(err.message, 'error');
                } finally {
                    btn.disabled = false;
                    btn.textContent = 'Run Cycle';
                }
            });
        });

        container.querySelectorAll('.btn-posts').forEach(btn => {
            btn.addEventListener('click', () => {
                const id = btn.dataset.id;
                renderPosts(container, id);
            });
        });

    } catch (err) {
        showNotification('Failed to load brands', 'error');
    }
}

async function renderBrandWizard(container) {
    document.getElementById('page-title').textContent = 'Create Brand Wizard';
    container.className = '';
    container.innerHTML = `
        <div class="glass card animate-fade" style="max-width: 600px; margin: 0 auto">
            <form id="brand-form">
                <div class="input-group">
                    <label>Brand ID (Unique Slug)</label>
                    <input type="text" id="b-id" placeholder="my_brand" required>
                </div>
                <div class="input-group">
                    <label>Brand Name</label>
                    <input type="text" id="b-name" placeholder="Nebula AI" required>
                </div>
                <div class="input-group">
                    <label>Industry</label>
                    <input type="text" id="b-industry" placeholder="SaaS, FinTech, etc." required>
                </div>
                <div class="input-group">
                    <label>Brand Voice</label>
                    <input type="text" id="b-voice" placeholder="Professional, visionary, yet pragmatic" required>
                </div>
                <div class="input-group">
                    <label>Target Audience</label>
                    <input type="text" id="b-audience" placeholder="CTOs, Developers, Healthcare Managers" required>
                </div>
                <div class="input-group">
                    <label>Schedule Interval (Hours)</label>
                    <input type="number" id="b-interval" value="4" min="1">
                </div>
                <div class="brand-actions" style="margin-top: 2rem">
                    <button type="submit" class="btn-primary">Save Brand</button>
                    <button type="button" id="cancel-wizard" class="btn-secondary">Cancel</button>
                </div>
            </form>
        </div>
    `;

    document.getElementById('cancel-wizard').addEventListener('click', () => renderBrands(container));
    document.getElementById('brand-form').addEventListener('submit', async (e) => {
        e.preventDefault();
        const brand = {
            id: document.getElementById('b-id').value,
            name: document.getElementById('b-name').value,
            industry: document.getElementById('b-industry').value,
            voice: document.getElementById('b-voice').value,
            target_audience: document.getElementById('b-audience').value,
            schedule_interval_hours: parseInt(document.getElementById('b-interval').value)
        };

        try {
            await request('/brands', 'POST', brand);
            showNotification('Brand created successfully!', 'success');
            renderBrands(container);
        } catch (err) {
            showNotification(err.message, 'error');
        }
    });
}

async function renderPosts(container, brandID) {
    document.getElementById('page-title').textContent = `Posts: ${brandID}`;
    container.className = 'animate-fade';
    container.innerHTML = '<div class="loader"></div>';

    try {
        const posts = await request(`/brands/${brandID}/posts`);
        if (posts.length === 0) {
            container.innerHTML = '<div class="glass card">No posts generated yet.</div>';
        } else {
            container.innerHTML = `
                <table style="width: 100%; border-collapse: collapse;">
                    <thead>
                        <tr style="text-align: left; border-bottom: 1px solid var(--glass-border)">
                            <th style="padding: 1rem">Date</th>
                            <th style="padding: 1rem">Topic</th>
                            <th style="padding: 1rem">Performance</th>
                            <th style="padding: 1rem">Actions</th>
                        </tr>
                    </thead>
                    <tbody>
                        ${posts.map(post => `
                            <tr style="border-bottom: 1px solid rgba(255,255,255,0.05)">
                                <td style="padding: 1rem">${new Date(post.created_at).toLocaleDateString()}</td>
                                <td style="padding: 1rem">${post.topic}</td>
                                <td style="padding: 1rem">👍 ${post.analytics.likes} | 🔄 ${post.analytics.shares}</td>
                                <td style="padding: 1rem"><a href="#" style="color: var(--primary-light)">View Detail</a></td>
                            </tr>
                        `).join('')}
                    </tbody>
                </table>
            `;
        }

        const backBtn = document.createElement('button');
        backBtn.className = 'btn-secondary';
        backBtn.style.marginTop = '2rem';
        backBtn.textContent = '← Back to Brands';
        backBtn.onclick = () => renderBrands(container);
        container.appendChild(backBtn);

    } catch (err) {
        showNotification('Failed to load posts', 'error');
    }
}

// --- Notifications ---

function showNotification(message, type = 'success') {
    const container = document.getElementById('toast-container');
    const toast = document.createElement('div');
    toast.className = `toast ${type}`;
    toast.textContent = message;
    container.appendChild(toast);

    setTimeout(() => {
        toast.style.opacity = '0';
        setTimeout(() => toast.remove(), 300);
    }, 4000);
}
