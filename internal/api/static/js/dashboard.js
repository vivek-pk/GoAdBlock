// Tailwind configuration
tailwind.config = {
  theme: {
    extend: {
      colors: {
        'tva': {
          'amber': '#FF8B28',
          'orange': '#FF6B00',
          'brown': '#794B28',
          'dark': '#251F17',
          'tan': '#D6BC97',
          'cream': '#F5E9D7',
          'black': '#1A1512',
        },
      },
      fontFamily: {
        'serif': ['"DM Serif Display"', 'serif'],
        'mono': ['"B612 Mono"', 'monospace'],
      },
    },
  },
};

// Main dashboard functionality
function dashboard() {
  // Define startTime outside the return object
  const startTime = new Date();
  let chart = null;

  // Get saved theme first thing
  const savedTheme = localStorage.getItem('theme') || 'tva';

  // Apply theme to body immediately (don't wait for Alpine)
  document.body.classList.add(`theme-${savedTheme}`);

  return {
    status: 'running',
    metrics: {
      totalQueries: 0,
      blockedQueries: 0,
      cacheHits: 0,
      cacheMisses: 0,
    },
    recentQueries: [],
    hourlyStats: {
      labels: [],
      requests: [],
      blocks: [],
    },
    clientStats: [],
    currentPage: 'dashboard',
    currentTheme: savedTheme,
    uptimeTick: 0, // Property for Alpine to track

    init() {
      // Apply theme on initialization
      this.applyTheme();

      this.initChart();
      this.fetchData();

      // Update data every 2 seconds
      setInterval(() => this.fetchData(), 2000);

      // Add a dedicated timer for updating the uptime display every second
      setInterval(() => {
        this.uptimeTick++; // This will trigger Alpine to re-evaluate the formatUptime()
      }, 1000);

      // Determine current page from URL
      const path = window.location.pathname;
      if (path.includes('/blocklists')) {
        this.currentPage = 'blocklists';
      } else if (path.includes('/settings')) {
        this.currentPage = 'settings';
      } else if (path.includes('/about')) {
        this.currentPage = 'about';
      } else {
        this.currentPage = 'dashboard';
      }
    },

    toggleTheme() {
      // Change theme with transition
      document.body.classList.add('theme-transitioning');

      setTimeout(() => {
        this.currentTheme = this.currentTheme === 'tva' ? 'cockpit' : 'tva';
        localStorage.setItem('theme', this.currentTheme);

        // Apply the theme
        document.body.classList.remove('theme-tva', 'theme-cockpit');
        document.body.classList.add(`theme-${this.currentTheme}`);

        // Recreate chart with new theme
        if (this.currentPage === 'dashboard' && chart) {
          chart.destroy();
          this.initChart();
        }

        // After theme is applied, remove the transitioning class
        setTimeout(() => {
          document.body.classList.remove('theme-transitioning');
        }, 500);
      }, 100);
    },

    applyTheme() {
      document.body.classList.remove('theme-tva', 'theme-cockpit');
      document.body.classList.add(`theme-${this.currentTheme}`);
    },

    formatUptime() {
      const now = new Date();
      const diff = now - startTime; // Use the constant from closure

      const days = Math.floor(diff / 86400000); // days
      const hours = Math.floor((diff % 86400000) / 3600000); // hours
      const minutes = Math.floor((diff % 3600000) / 60000); // minutes
      const seconds = Math.floor((diff % 60000) / 1000); // seconds

      let uptimeString = '';

      if (days > 0) {
        uptimeString += `${days}D `;
      }

      return (
        uptimeString +
        `${hours.toString().padStart(2, '0')}:${minutes
          .toString()
          .padStart(2, '0')}:${seconds.toString().padStart(2, '0')}`
      );
    },

    getUptimePercentage() {
      const now = new Date();
      const diff = now - startTime; // Use the constant from closure
      const minutesPassed = Math.floor((diff % 3600000) / 60000); // minutes within current hour
      return (minutesPassed / 60) * 100;
    },

    initChart() {
      if (!window.Chart) {
        console.error('Chart.js not loaded');
        return;
      }

      const ctx = document.getElementById('statsChart').getContext('2d');
      if (!ctx) {
        console.error('Could not find stats chart context');
        return;
      }

      // Set colors based on current theme
      let primaryColor, secondaryColor;
      let primaryGradient, secondaryGradient;
      let chartBackgroundColor;

      if (this.currentTheme === 'tva') {
        primaryColor = '#FF6B00';
        secondaryColor = '#794B28';
        chartBackgroundColor = '#1A1512'; // Dark background for TVA theme

        // Custom gradient for the sacred timeline
        primaryGradient = ctx.createLinearGradient(0, 0, 0, 300);
        primaryGradient.addColorStop(0, 'rgba(255, 107, 0, 0.8)');
        primaryGradient.addColorStop(1, 'rgba(255, 139, 40, 0.3)');

        // Custom gradient for the variant timeline
        secondaryGradient = ctx.createLinearGradient(0, 0, 0, 300);
        secondaryGradient.addColorStop(0, 'rgba(121, 75, 40, 0.8)');
        secondaryGradient.addColorStop(1, 'rgba(121, 75, 40, 0.3)');
      } else {
        // Cockpit theme
        primaryColor = '#10B981';
        secondaryColor = '#38BDF8';
        chartBackgroundColor = '#051826'; // Already dark for cockpit theme

        // Custom gradient for cockpit primary line
        primaryGradient = ctx.createLinearGradient(0, 0, 0, 300);
        primaryGradient.addColorStop(0, 'rgba(16, 185, 129, 0.8)');
        primaryGradient.addColorStop(1, 'rgba(16, 185, 129, 0.2)');

        // Custom gradient for cockpit secondary line
        secondaryGradient = ctx.createLinearGradient(0, 0, 0, 300);
        secondaryGradient.addColorStop(0, 'rgba(56, 189, 248, 0.8)');
        secondaryGradient.addColorStop(1, 'rgba(56, 189, 248, 0.2)');
      }

      Chart.defaults.color =
        this.currentTheme === 'tva' ? '#794B28' : '#e0f2f1';
      Chart.defaults.font.family = "'B612 Mono', monospace";

      const options = {
        responsive: true,
        maintainAspectRatio: false,
        animation: {
          duration: 1500,
          easing: 'easeOutQuart',
        },
        plugins: {
          legend: {
            position: 'top',
            labels: {
              boxWidth: 15,
              usePointStyle: true,
              pointStyle: 'rectRot',
              padding: 20,
              font: {
                family: "'B612 Mono', monospace",
                size: 12,
              },
              color: this.currentTheme === 'tva' ? '#794B28' : '#e0f2f1',
            },
          },
          tooltip: {
            backgroundColor:
              this.currentTheme === 'tva'
                ? 'rgba(214, 188, 151, 0.9)'
                : 'rgba(16, 185, 129, 0.8)',
            titleFont: {
              family: "'DM Serif Display', serif",
              size: 14,
              weight: 'bold',
            },
            bodyFont: {
              family: "'B612 Mono', monospace",
              size: 12,
            },
            borderColor: primaryColor,
            borderWidth: 2,
            titleColor: this.currentTheme === 'tva' ? '#251F17' : '#e0f2f1',
            bodyColor: this.currentTheme === 'tva' ? '#251F17' : '#e0f2f1',
            padding: 12,
            boxPadding: 5,
            displayColors: true,
            callbacks: {
              title: function (tooltipItems) {
                return (
                  (this.currentTheme === 'tva'
                    ? 'TIMELINE POINT: '
                    : 'SIGNAL DATA: ') + tooltipItems[0].label
                );
              }.bind(this),
            },
          },
        },
        scales: {
          y: {
            beginAtZero: true,
            ticks: {
              font: {
                family: "'B612 Mono', monospace",
                size: 10,
              },
              color: this.currentTheme === 'tva' ? '#D6BC97' : '#e0f2f1', // Lighter color for better contrast
            },
            grid: {
              color:
                this.currentTheme === 'tva'
                  ? 'rgba(121, 75, 40, 0.15)'
                  : 'rgba(16, 185, 129, 0.15)',
              drawBorder: false,
            },
            title: {
              display: true,
              text:
                this.currentTheme === 'tva'
                  ? 'TIMELINE MAGNITUDE'
                  : 'SIGNAL STRENGTH',
              font: {
                family: "'B612 Mono', monospace",
                size: 10,
                weight: 'bold',
              },
              color: this.currentTheme === 'tva' ? '#D6BC97' : '#e0f2f1', // Lighter color for better contrast
            },
          },
          x: {
            grid: {
              color:
                this.currentTheme === 'tva'
                  ? 'rgba(121, 75, 40, 0.15)'
                  : 'rgba(16, 185, 129, 0.1)',
              drawBorder: false,
            },
            ticks: {
              font: {
                family: "'B612 Mono', monospace",
                size: 10,
              },
              color: this.currentTheme === 'tva' ? '#D6BC97' : '#e0f2f1', // Lighter color for better contrast
            },
            title: {
              display: true,
              text:
                this.currentTheme === 'tva'
                  ? 'TEMPORAL COORDINATES'
                  : 'SURVEILLANCE INTERVAL',
              font: {
                family: "'B612 Mono', monospace",
                size: 10,
                weight: 'bold',
              },
              color: this.currentTheme === 'tva' ? '#D6BC97' : '#e0f2f1', // Lighter color for better contrast
            },
          },
        },
        elements: {
          line: {
            borderWidth: 3,
            borderCapStyle: 'round',
          },
          point: {
            hitRadius: 10,
            hoverRadius: 8,
            hoverBorderWidth: 2,
          },
        },
      };

      // Set darker grid lines for cockpit theme
      if (this.currentTheme === 'cockpit') {
        options.scales.y.grid.color = 'rgba(16, 185, 129, 0.1)';
        options.scales.x.grid.color = 'rgba(16, 185, 129, 0.1)';
        options.scales.y.ticks.color = '#10b981';
        options.scales.x.ticks.color = '#10b981';
      }

      chart = new Chart(ctx, {
        type: 'line',
        data: {
          labels: [],
          datasets: [
            {
              label:
                this.currentTheme === 'tva'
                  ? 'Sacred Timeline'
                  : 'Primary Signals',
              borderColor: primaryColor,
              backgroundColor: primaryGradient,
              borderWidth: 3,
              pointBackgroundColor: primaryColor,
              pointBorderColor:
                this.currentTheme === 'tva' ? '#1A1512' : '#0a0e17',
              pointRadius: 6,
              pointHoverRadius: 8,
              data: [],
              fill: true,
              tension: 0.2,
              pointStyle: this.currentTheme === 'tva' ? 'rectRot' : 'circle',
              borderDash: [],
            },
            {
              label:
                this.currentTheme === 'tva'
                  ? 'Variant Branches'
                  : 'Secondary Signals',
              borderColor: secondaryColor,
              backgroundColor: secondaryGradient,
              borderWidth: 2,
              pointBackgroundColor: secondaryColor,
              pointBorderColor:
                this.currentTheme === 'tva' ? '#1A1512' : '#0a0e17',
              pointRadius: 6,
              pointHoverRadius: 8,
              data: [],
              fill: true,
              tension: 0.2,
              borderDash: [5, 5],
              pointStyle: this.currentTheme === 'tva' ? 'rectRot' : 'triangle',
            },
          ],
          options: options,
        },
      });

      // Also update the chart container CSS
      const chartContainer = document.querySelector('.chart-container');
      if (chartContainer) {
        if (this.currentTheme === 'tva') {
          chartContainer.style.backgroundColor = '#1A1512'; // Dark background for TVA theme
          chartContainer.style.border = '1px solid #794B28';
        } else {
          chartContainer.style.backgroundColor = '#051826'; // Dark for cockpit theme
          chartContainer.style.border = '1px solid #38bdf8';
        }
      }

      // Add chart annotation line
      try {
        if (chartContainer) {
          // Remove existing timeline divider if any
          const existingDivider =
            chartContainer.querySelector('.timeline-divider');
          if (existingDivider) {
            existingDivider.remove();
          }

          const timelineDivider = document.createElement('div');
          timelineDivider.className = 'timeline-divider';
          chartContainer.appendChild(timelineDivider);
        }
      } catch (e) {
        console.error('Could not add timeline divider:', e);
      }
    },

    async fetchData() {
      try {
        // Fetch metrics
        const metricsResponse = await fetch('/api/v1/metrics');
        if (!metricsResponse.ok) {
          throw new Error('Metrics fetch failed');
        }
        const metricsData = await metricsResponse.json();

        // Update metrics
        this.metrics = {
          totalQueries: metricsData.totalQueries || 0,
          blockedQueries: metricsData.blockedQueries || 0,
          cacheHits: metricsData.cacheHits || 0,
          cacheMisses: metricsData.cacheMisses || 0,
        };

        // Fetch recent queries
        const queriesResponse = await fetch('/api/v1/queries');
        if (!queriesResponse.ok) {
          throw new Error('Queries fetch failed');
        }
        const queriesData = await queriesResponse.json();
        this.recentQueries = (queriesData.queries || []).map((q) => ({
          ...q,
          time: new Date(q.timestamp).toLocaleTimeString(),
        }));

        // Update status
        const statusResponse = await fetch('/api/v1/status');
        if (!statusResponse.ok) {
          throw new Error('Status fetch failed');
        }
        const statusData = await statusResponse.json();
        this.status = statusData.status;

        // Fetch hourly stats
        const statsResponse = await fetch('/api/v1/stats/hourly');
        if (!statsResponse.ok) {
          throw new Error('Hourly stats fetch failed');
        }
        const statsData = await statsResponse.json();

        // Fetch client stats
        const clientStatsResponse = await fetch('/api/v1/clients');
        if (!clientStatsResponse.ok) {
          throw new Error('Client stats fetch failed');
        }
        const clientStatsData = await clientStatsResponse.json();
        this.clientStats = clientStatsData.clients.map((client) => ({
          ...client,
          lastSeen: new Date(client.lastSeen).toLocaleString(),
        }));

        // Update chart
        if (chart) {
          chart.data.labels = statsData.hours;
          chart.data.datasets[0].data = statsData.requests;
          chart.data.datasets[1].data = statsData.blocks;
          chart.update();
        }
      } catch (error) {
        console.error('Failed to fetch data:', error);
        this.status = 'stopped';
      }
    },
  };
}
