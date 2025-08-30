const apiUrl = 'https://your-api-gateway-url/prod/findings'; // Replace with actual API Gateway URL

async function fetchFindings() {
    try {
        const response = await fetch(apiUrl);
        const findings = await response.json();
        displayFindings(findings);
    } catch (error) {
        console.error('Error fetching findings:', error);
    }
}

function displayFindings(findings) {
    const totalFindings = document.getElementById('total-findings');
    const highSeverity = document.getElementById('high-severity');
    const findingsBody = document.getElementById('findings-body');

    totalFindings.textContent = findings.length;
    highSeverity.textContent = findings.filter(f => f.severity === 'High').length;

    findingsBody.innerHTML = '';
    findings.forEach(finding => {
        const row = document.createElement('tr');
        row.innerHTML = `
            <td>${finding.resource_type}</td>
            <td>${finding.resource_id}</td>
            <td>${finding.issue}</td>
            <td class="severity-${finding.severity.toLowerCase()}">${finding.severity}</td>
            <td>${new Date(finding.timestamp).toLocaleString()}</td>
        `;
        findingsBody.appendChild(row);
    });
}

// Load findings on page load
window.onload = fetchFindings;