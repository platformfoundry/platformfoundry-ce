// API Base URL
const API_BASE = window.location.origin;

// State
let currentView = 'platforms';

// Initialize
document.addEventListener('DOMContentLoaded', () => {
    initializeNavigation();
    initializeForm();
    loadStats();
    refreshPlatforms();

    // Auto-refresh every 30 seconds
    setInterval(() => {
        loadStats();
        if (currentView === 'platforms') refreshPlatforms();
        if (currentView === 'jobs') refreshJobs();
    }, 30000);
});

// Navigation
function initializeNavigation() {
    const navBtns = document.querySelectorAll('.nav-btn');
    navBtns.forEach(btn => {
        btn.addEventListener('click', () => {
            const view = btn.dataset.view;
            switchView(view);
        });
    });
}

function switchView(view) {
    currentView = view;

    // Update nav buttons
    document.querySelectorAll('.nav-btn').forEach(btn => {
        btn.classList.toggle('active', btn.dataset.view === view);
    });

    // Update views
    document.querySelectorAll('.view').forEach(v => {
        v.classList.toggle('active', v.id === `${view}-view`);
    });

    // Load data for view
    if (view === 'platforms') refreshPlatforms();
    if (view === 'jobs') refreshJobs();
}

// Stats
async function loadStats() {
    try {
        const response = await fetch(`${API_BASE}/api/stats`);
        const data = await response.json();

        if (data.success) {
            document.getElementById('platformCount').textContent = data.data.platforms || 0;
            document.getElementById('jobCount').textContent = data.data.jobs || 0;
            document.getElementById('resourceCount').textContent = data.data.resources || 0;
        }
    } catch (error) {
        console.error('Failed to load stats:', error);
    }
}

// Platforms
async function refreshPlatforms() {
    try {
        const response = await fetch(`${API_BASE}/api/platforms`);
        const data = await response.json();

        const grid = document.getElementById('platformsGrid');

        if (data.success && data.data && data.data.length > 0) {
            grid.innerHTML = data.data.map(platform => createPlatformCard(platform)).join('');
        } else {
            grid.innerHTML = '<p class="empty-state">No platforms found. Create one to get started.</p>';
        }
    } catch (error) {
        console.error('Failed to load platforms:', error);
        document.getElementById('platformsGrid').innerHTML =
            '<p class="empty-state">Error loading platforms. Please try again.</p>';
    }
}

function createPlatformCard(platform) {
    return `
        <div class="platform-card">
            <div class="platform-name">${platform.name}</div>
            <div class="platform-meta">
                <span>${platform.type}</span>
                <span>${platform.resources} resources</span>
            </div>
            <div>
                <span class="platform-status status-${platform.status.toLowerCase()}">${platform.status}</span>
            </div>
            <div class="platform-actions">
                <button class="btn btn-sm" onclick="viewPlatform('${platform.name}')">View</button>
                <button class="btn btn-sm btn-danger" onclick="deletePlatform('${platform.name}')">Delete</button>
            </div>
        </div>
    `;
}

async function viewPlatform(name) {
    try {
        const response = await fetch(`${API_BASE}/api/platforms/${name}`);
        const data = await response.json();

        if (data.success) {
            alert(`Platform: ${name}\n${JSON.stringify(data.data, null, 2)}`);
        }
    } catch (error) {
        console.error('Failed to view platform:', error);
        alert('Failed to load platform details');
    }
}

async function deletePlatform(name) {
    if (!confirm(`Are you sure you want to delete platform "${name}"?`)) {
        return;
    }

    try {
        const response = await fetch(`${API_BASE}/api/platforms/${name}`, {
            method: 'DELETE'
        });
        const data = await response.json();

        if (data.success) {
            refreshPlatforms();
            loadStats();
        } else {
            alert('Failed to delete platform: ' + (data.error || 'Unknown error'));
        }
    } catch (error) {
        console.error('Failed to delete platform:', error);
        alert('Failed to delete platform');
    }
}

// Jobs
async function refreshJobs() {
    try {
        const response = await fetch(`${API_BASE}/api/jobs`);
        const data = await response.json();

        const list = document.getElementById('jobsList');

        if (data.success && data.data && data.data.length > 0) {
            list.innerHTML = data.data.map(job => createJobItem(job)).join('');
        } else {
            list.innerHTML = '<p class="empty-state">No active jobs.</p>';
        }
    } catch (error) {
        console.error('Failed to load jobs:', error);
        document.getElementById('jobsList').innerHTML =
            '<p class="empty-state">Error loading jobs. Please try again.</p>';
    }
}

function createJobItem(job) {
    return `
        <div class="job-item">
            <div class="job-header">
                <span class="job-id">${job.id}</span>
                <span class="platform-status status-${job.status.toLowerCase()}">${job.status}</span>
            </div>
            <div>
                <strong>Type:</strong> ${job.type}
            </div>
            <div class="job-progress">
                <div class="job-progress-bar" style="width: ${job.progress}%"></div>
            </div>
            <div style="margin-top: 10px; font-size: 12px; color: #7f8c8d;">
                Progress: ${job.progress}%
            </div>
        </div>
    `;
}

// Form
function initializeForm() {
    const form = document.getElementById('createPlatformForm');
    form.addEventListener('submit', async (e) => {
        e.preventDefault();

        const formData = new FormData(form);
        const platform = {
            name: formData.get('name'),
            type: formData.get('type'),
            metadata: {
                region: formData.get('region')
            }
        };

        try {
            const response = await fetch(`${API_BASE}/api/platforms`, {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json'
                },
                body: JSON.stringify(platform)
            });

            const data = await response.json();

            if (data.success) {
                alert('Platform created successfully!');
                form.reset();
                switchView('platforms');
                loadStats();
            } else {
                alert('Failed to create platform: ' + (data.error || 'Unknown error'));
            }
        } catch (error) {
            console.error('Failed to create platform:', error);
            alert('Failed to create platform');
        }
    });
}

function resetForm() {
    document.getElementById('createPlatformForm').reset();
}

// YAML Editor
async function validateYAML() {
    const yaml = document.getElementById('yamlEditor').value;
    const resultDiv = document.getElementById('validationResult');

    if (!yaml.trim()) {
        resultDiv.innerHTML = '<p style="color: #e74c3c;">Please enter YAML content</p>';
        return;
    }

    try {
        const response = await fetch(`${API_BASE}/api/validate`, {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json'
            },
            body: JSON.stringify({ yaml })
        });

        const data = await response.json();

        if (data.success && data.data.valid) {
            resultDiv.innerHTML = `
                <div style="color: #27ae60;">
                    <strong>✓ Valid YAML</strong>
                    <p style="margin-top: 10px;">No errors found</p>
                </div>
            `;
        } else {
            const errors = data.data.errors || [];
            resultDiv.innerHTML = `
                <div style="color: #e74c3c;">
                    <strong>✗ Validation Errors</strong>
                    <ul style="margin-top: 10px; padding-left: 20px;">
                        ${errors.map(err => `<li>${err}</li>`).join('')}
                    </ul>
                </div>
            `;
        }
    } catch (error) {
        console.error('Validation failed:', error);
        resultDiv.innerHTML = '<p style="color: #e74c3c;">Validation failed. Please try again.</p>';
    }
}

async function applyYAML() {
    const yaml = document.getElementById('yamlEditor').value;

    if (!yaml.trim()) {
        alert('Please enter YAML content');
        return;
    }

    if (!confirm('Are you sure you want to apply this configuration?')) {
        return;
    }

    try {
        const response = await fetch(`${API_BASE}/api/apply`, {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json'
            },
            body: JSON.stringify({ yaml })
        });

        const data = await response.json();

        if (data.success) {
            alert(`Apply started! Job ID: ${data.data.job_id}`);
            switchView('jobs');
        } else {
            alert('Failed to apply: ' + (data.error || 'Unknown error'));
        }
    } catch (error) {
        console.error('Apply failed:', error);
        alert('Failed to apply configuration');
    }
}
