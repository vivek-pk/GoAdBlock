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
  // Define startTime outside the return object to ensure it's created at initialization time
  const startTime = new Date();
  let chart = null;

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
    uptimeTick: 0, // Property for Alpine to track

    init() {
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
      const ctx = document.getElementById('statsChart').getContext('2d');

      // Custom gradient for the sacred timeline
      const sacredTimelineGradient = ctx.createLinearGradient(0, 0, 0, 300);
      sacredTimelineGradient.addColorStop(0, 'rgba(255, 107, 0, 0.8)');
      sacredTimelineGradient.addColorStop(1, 'rgba(255, 139, 40, 0.3)');

      // Custom gradient for the variant timeline
      const variantTimelineGradient = ctx.createLinearGradient(0, 0, 0, 300);
      variantTimelineGradient.addColorStop(0, 'rgba(121, 75, 40, 0.8)');
      variantTimelineGradient.addColorStop(1, 'rgba(121, 75, 40, 0.3)');

      Chart.defaults.color = '#794B28';
      Chart.defaults.font.family = "'B612 Mono', monospace";

      chart = new Chart(ctx, {
        type: 'line',
        data: {
          labels: [],
          datasets: [
            {
              label: 'Sacred Timeline',
              borderColor: '#FF6B00',
              backgroundColor: sacredTimelineGradient,
              borderWidth: 3,
              pointBackgroundColor: '#FF6B00',
              pointBorderColor: '#1A1512',
              pointRadius: 6,
              pointHoverRadius: 8,
              data: [],
              fill: true,
              tension: 0.2,
              pointStyle: 'rectRot',
              borderDash: [],
            },
            {
              label: 'Variant Branches',
              borderColor: '#794B28',
              backgroundColor: variantTimelineGradient,
              borderWidth: 2,
              pointBackgroundColor: '#794B28',
              pointBorderColor: '#1A1512',
              pointRadius: 6,
              pointHoverRadius: 8,
              data: [],
              fill: true,
              tension: 0.2,
              borderDash: [5, 5],
              pointStyle: 'rectRot',
            },
          ],
        },
        options: {
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
                color: '#794B28',
              },
            },
            tooltip: {
              backgroundColor: 'rgba(214, 188, 151, 0.9)',
              titleFont: {
                family: "'DM Serif Display', serif",
                size: 14,
                weight: 'bold',
              },
              bodyFont: {
                family: "'B612 Mono', monospace",
                size: 12,
              },
              borderColor: '#FF6B00',
              borderWidth: 2,
              titleColor: '#251F17',
              bodyColor: '#251F17',
              padding: 12,
              boxPadding: 5,
              displayColors: true,
              callbacks: {
                title: function (tooltipItems) {
                  return 'TIMELINE POINT: ' + tooltipItems[0].label;
                },
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
                color: '#794B28',
              },
              grid: {
                color: 'rgba(121, 75, 40, 0.15)',
                drawBorder: false,
              },
              title: {
                display: true,
                text: 'TIMELINE MAGNITUDE',
                font: {
                  family: "'B612 Mono', monospace",
                  size: 10,
                  weight: 'bold',
                },
                color: '#794B28',
              },
            },
            x: {
              grid: {
                color: 'rgba(121, 75, 40, 0.1)',
                drawBorder: false,
              },
              ticks: {
                font: {
                  family: "'B612 Mono', monospace",
                  size: 10,
                },
                color: '#794B28',
              },
              title: {
                display: true,
                text: 'TEMPORAL COORDINATES',
                font: {
                  family: "'B612 Mono', monospace",
                  size: 10,
                  weight: 'bold',
                },
                color: '#794B28',
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
        },
      });

      // Add chart annotation to simulate the "red line" in TVA animations
      const timelineDivider = document.createElement('div');
      timelineDivider.className = 'timeline-divider';
      document.querySelector('.chart-container').appendChild(timelineDivider);
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
