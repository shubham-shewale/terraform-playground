// AWS CSPM Monitor Dashboard JavaScript
class CSPMDashboard {
    constructor() {
        // Use environment variable or fallback to same origin
        this.apiBaseUrl = window.CSPM_API_URL || window.location.origin + '/prod';
        this.findings = [];
        this.severityChart = null;

        this.init();
    }

    init() {
        this.bindEvents();
        this.loadSummary();
        this.loadFindings();
    }

    bindEvents() {
        // Refresh button
        document.getElementById('refresh-btn').addEventListener('click', () => {
            this.loadSummary();
            this.loadFindings();
        });

        // Severity filter
        document.getElementById('severity-filter').addEventListener('change', () => {
            this.loadFindings();
        });

        // Limit input
        document.getElementById('limit-input').addEventListener('change', () => {
            this.loadFindings();
        });
    }

    async makeApiCall(endpoint, params = {}) {
        try {
            const url = new URL(`${this.apiBaseUrl}${endpoint}`);
            Object.keys(params).forEach(key => {
                if (params[key]) {
                    url.searchParams.append(key, params[key]);
                }
            });

            const response = await fetch(url, {
                method: 'GET',
                headers: {
                    'Content-Type': 'application/json',
                },
            });

            if (!response.ok) {
                throw new Error(`HTTP error! status: ${response.status}`);
            }

            const data = await response.json();
            return data;
        } catch (error) {
            console.error('API call failed:', error);
            this.showError(`Failed to load data: ${error.message}`);
            return null;
        }
    }

    async loadSummary() {
        const summary = await this.makeApiCall('/summary');
        if (summary && summary.success) {
            this.updateSummaryCards(summary.data);
            this.updateChart(summary.data.severity_breakdown);
        }
    }

    async loadFindings() {
        const severity = document.getElementById('severity-filter').value;
        const limit = document.getElementById('limit-input').value;

        const params = {};
        if (severity) params.severity = severity;
        if (limit) params.limit = limit;

        const response = await this.makeApiCall('/findings', params);
        if (response && response.success) {
            this.findings = response.data;
            this.updateFindingsTable();
        }
    }

    updateSummaryCards(summary) {
        document.getElementById('total-findings').textContent = summary.total_findings || 0;
        document.getElementById('critical-count').textContent = summary.severity_breakdown?.CRITICAL || 0;
        document.getElementById('high-count').textContent = summary.severity_breakdown?.HIGH || 0;
        document.getElementById('medium-count').textContent = summary.severity_breakdown?.MEDIUM || 0;
    }

    updateChart(severityBreakdown) {
        const ctx = document.getElementById('severityChart').getContext('2d');

        if (this.severityChart) {
            this.severityChart.destroy();
        }

        const labels = Object.keys(severityBreakdown || {});
        const data = Object.values(severityBreakdown || {});

        this.severityChart = new Chart(ctx, {
            type: 'doughnut',
            data: {
                labels: labels,
                datasets: [{
                    data: data,
                    backgroundColor: [
                        '#dc3545', // Critical - Red
                        '#fd7e14', // High - Orange
                        '#ffc107', // Medium - Yellow
                        '#28a745', // Low - Green
                        '#6c757d'  // Informational - Gray
                    ],
                    borderWidth: 2,
                    borderColor: '#ffffff'
                }]
            },
            options: {
                responsive: true,
                maintainAspectRatio: false,
                plugins: {
                    legend: {
                        position: 'bottom',
                    },
                    tooltip: {
                        callbacks: {
                            label: function(context) {
                                const total = context.dataset.data.reduce((a, b) => a + b, 0);
                                const percentage = ((context.parsed / total) * 100).toFixed(1);
                                return `${context.label}: ${context.parsed} (${percentage}%)`;
                            }
                        }
                    }
                }
            }
        });
    }

    updateFindingsTable() {
        const tbody = document.getElementById('findings-tbody');

        if (this.findings.length === 0) {
            tbody.innerHTML = '<tr><td colspan="6" class="no-data">No findings found</td></tr>';
            return;
        }

        tbody.innerHTML = this.findings.map(finding => {
            const severityClass = finding.severity?.toLowerCase() || 'unknown';
            const timestamp = new Date(finding.timestamp).toLocaleString();

            return `
                <tr>
                    <td><span class="severity-badge ${severityClass}">${finding.severity || 'UNKNOWN'}</span></td>
                    <td class="title-cell" title="${finding.description || ''}">${finding.title || 'No Title'}</td>
                    <td>${finding.resource_type || 'Unknown'}</td>
                    <td>${finding.account_id || 'Unknown'}</td>
                    <td>${finding.region || 'Unknown'}</td>
                    <td>${timestamp}</td>
                </tr>
            `;
        }).join('');
    }

    showError(message) {
        // Create error notification
        const errorDiv = document.createElement('div');
        errorDiv.className = 'error-notification';
        errorDiv.innerHTML = `
            <span class="error-icon">⚠️</span>
            <span class="error-message">${message}</span>
            <button class="error-close" onclick="this.parentElement.remove()">×</button>
        `;

        // Add to page
        document.body.appendChild(errorDiv);

        // Auto-remove after 5 seconds
        setTimeout(() => {
            if (errorDiv.parentElement) {
                errorDiv.remove();
            }
        }, 5000);
    }
}

// Initialize dashboard when page loads
document.addEventListener('DOMContentLoaded', () => {
    window.dashboard = new CSPMDashboard();

    // Auto-refresh every 5 minutes
    setInterval(() => {
        window.dashboard.loadSummary();
        window.dashboard.loadFindings();
    }, 5 * 60 * 1000);
});

// Handle API Gateway URL configuration
// This would be replaced with the actual API Gateway URL during deployment
if (window.location.hostname === 'localhost' || window.location.hostname === '127.0.0.1') {
    // For local development, you might need to configure a proxy or use a different URL
    console.log('Running in development mode');
}