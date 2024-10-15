package exporter

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/google/go-github/v66/github"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/promhippie/github_exporter/pkg/config"
	"github.com/promhippie/github_exporter/pkg/store"
)

// BranchCollector collects metrics about the branch of a repo.
type BranchCollector struct {
	client   *github.Client
	logger   *slog.Logger
	db       store.Store
	failures *prometheus.CounterVec
	duration *prometheus.HistogramVec
	config   config.Target

	Protected   *prometheus.Desc
	LastUpdated *prometheus.Desc
}

// NewBranchCollector returns a new BranchCollector.
func NewBranchCollector(logger *slog.Logger, client *github.Client, db store.Store, failures *prometheus.CounterVec, duration *prometheus.HistogramVec, cfg config.Target) *BranchCollector {
	if failures != nil {
		failures.WithLabelValues("branch").Add(0)
	}

	labels := []string{"owner", "repo", "branch", "last_updated"}
	return &BranchCollector{
		client:   client,
		logger:   logger.With("collector", "branch"),
		db:       db,
		failures: failures,
		duration: duration,
		config:   cfg,

		Protected: prometheus.NewDesc(
			"github_repo_branch_protected",
			"Aasdadasdasd",
			labels,
			nil,
		),
		LastUpdated: prometheus.NewDesc(
			"github_repo_branch_last_updated",
			"Aasdadasdasd",
			labels,
			nil,
		),
	}
}

// Metrics simply returns the list metric descriptors for generating a documentation.
func (c *BranchCollector) Metrics() []*prometheus.Desc {
	return []*prometheus.Desc{
		c.Protected,
		c.LastUpdated,
	}
}

// Describe sends the super-set of all possible descriptors of metrics collected by this Collector.
func (c *BranchCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.Protected
	ch <- c.LastUpdated
}

var lastCommit map[string]
// branch name
// commit timestamp
// count

// Collect is called by the Prometheus registry when collecting metrics.
func (c *BranchCollector) Collect(ch chan<- prometheus.Metric) {
	for _, name := range c.config.Repos.Value() {
		n := strings.Split(name, "/")

		if len(n) != 2 {
			c.logger.Error("Invalid repo name",
				"name", name,
			)

			c.failures.WithLabelValues("repo").Inc()
			continue
		}

		owner, repo := n[0], n[1]

		ctx, cancel := context.WithTimeout(context.Background(), c.config.Timeout)
		defer cancel()

		now := time.Now()
		records, err := reposByOwnerAndName(ctx, c.client, owner, repo, c.config.PerPage)
		c.duration.WithLabelValues("repo").Observe(time.Since(now).Seconds())

		if err != nil {
			c.logger.Error("Failed to fetch repos",
				"name", name,
				"err", err,
			)

			c.failures.WithLabelValues("repo").Inc()
			continue
		}

		c.logger.Debug("Fetched repos",
			"count", len(records),
			"duration", time.Since(now),
		)

		for _, record := range records {
			//masterBranches = append(masterBranches, record.GetMasterBranch())
			branches, err := branchByOwnerRepoAndName(ctx, c.client, owner, repo, record.GetDefaultBranch())

			if err != nil {
				c.logger.Error("Failed to fetch branch",
					"name", name,
					"err", err,
				)

				c.failures.WithLabelValues("branch").Inc()
				continue
			}

			for _, branch := range branches {

				lastCommit[fmt.Sprintf("%s-%s", repo, branch.GetName())] =

				labels := []string{
					owner,
					record.GetName(),
					branch.GetName(),
					branch.Commit.Commit.Author.GetDate().String(),
					branch.Commit.GetStats().Total
				}

				ch <- prometheus.MustNewConstMetric(
					c.Protected,
					prometheus.GaugeValue,
					boolToFloat64(branch.GetProtected()),
					labels...,
				)

				ch <- prometheus.MustNewConstMetric(
					c.LastUpdated,
					prometheus.GaugeValue,
					float64(branch.Commit.Commit.Author.GetDate().Unix()),
					labels...,
				)
			}
		}
	}
}
