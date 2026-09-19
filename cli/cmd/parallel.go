package cmd

import (
	"runtime"
	"sync"

	"github.com/drewstinnett/sourceseedy/internal/project"
)

// statusWorkers is how many repos are checked at once. Each check runs git, so
// this follows the number of CPUs, but stays within limits so that a big tree
// can't start hundreds of processes together, or crawl on a small machine
func statusWorkers() int {
	return min(max(runtime.NumCPU(), 4), 16)
}

// eachProject runs fn for every project, a few at a time, and returns the
// results in the same order as projects
func eachProject[T any](projects []project.Project, fn func(project.Project) T) []T {
	results := make([]T, len(projects))
	sem := make(chan struct{}, statusWorkers())
	var wg sync.WaitGroup
	for i, p := range projects {
		sem <- struct{}{}
		wg.Go(func() {
			defer func() { <-sem }()
			results[i] = fn(p)
		})
	}
	wg.Wait()
	return results
}
