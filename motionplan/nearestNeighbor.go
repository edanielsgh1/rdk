package motionplan

import (
	"context"
	"math"
	"sort"
	"sync"

	"go.viam.com/utils"

	"go.viam.com/rdk/referenceframe"
)

const neighborsBeforeParallelization = 1000

type neighborManager struct {
	nnKeys    chan node
	neighbors chan *neighbor
	nnLock    sync.RWMutex
	seedPos   []referenceframe.Input
	ready     bool
	nCPU      int
}

type neighbor struct {
	dist float64
	node node
}

//nolint:revive
func kNearestNeighbors(planOpts *PlannerOptions, rrtMap map[node]node, target []referenceframe.Input, neighborhoodSize int) []*neighbor {
	kNeighbors := neighborhoodSize
	if neighborhoodSize > len(rrtMap) {
		kNeighbors = len(rrtMap)
	}

	allCosts := make([]*neighbor, 0)
	for node := range rrtMap {
		_, dist := planOpts.DistanceFunc(&ConstraintInput{
			StartInput: target,
			EndInput:   node.Q(),
		})
		allCosts = append(allCosts, &neighbor{dist: dist, node: node})
	}
	sort.Slice(allCosts, func(i, j int) bool {
		return allCosts[i].dist < allCosts[j].dist
	})
	return allCosts[:kNeighbors]
}

func (nm *neighborManager) nearestNeighbor(
	ctx context.Context,
	planOpts *PlannerOptions,
	seed []referenceframe.Input,
	rrtMap map[node]node,
) node {
	if len(rrtMap) > neighborsBeforeParallelization && nm.nCPU > 1 {
		// If the map is large, calculate distances in parallel
		return nm.parallelNearestNeighbor(ctx, planOpts, seed, rrtMap)
	}
	bestDist := math.Inf(1)
	var best node
	for k := range rrtMap {
		_, dist := planOpts.DistanceFunc(&ConstraintInput{
			StartInput: seed,
			EndInput:   k.Q(),
		})
		if dist < bestDist {
			bestDist = dist
			best = k
		}
	}
	return best
}

func (nm *neighborManager) parallelNearestNeighbor(
	ctx context.Context,
	planOpts *PlannerOptions,
	seed []referenceframe.Input,
	rrtMap map[node]node,
) node {
	nm.ready = false
	nm.seedPos = seed
	nm.startNNworkers(ctx, planOpts)
	defer close(nm.nnKeys)
	defer close(nm.neighbors)

	for k := range rrtMap {
		nm.nnKeys <- k
	}
	nm.nnLock.Lock()
	nm.ready = true
	nm.nnLock.Unlock()
	var best node
	bestDist := math.Inf(1)
	returned := 0
	for returned < nm.nCPU {
		select {
		case <-ctx.Done():
			return nil
		default:
		}

		select {
		case nn := <-nm.neighbors:
			returned++
			if nn.dist < bestDist {
				bestDist = nn.dist
				best = nn.node
			}
		default:
		}
	}
	return best
}

func (nm *neighborManager) startNNworkers(ctx context.Context, planOpts *PlannerOptions) {
	nm.neighbors = make(chan *neighbor, nm.nCPU)
	nm.nnKeys = make(chan node, nm.nCPU)
	for i := 0; i < nm.nCPU; i++ {
		utils.PanicCapturingGo(func() {
			nm.nnWorker(ctx, planOpts)
		})
	}
}

func (nm *neighborManager) nnWorker(ctx context.Context, planOpts *PlannerOptions) {
	var best node
	bestDist := math.Inf(1)

	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		select {
		case k := <-nm.nnKeys:
			if k != nil {
				nm.nnLock.RLock()
				_, dist := planOpts.DistanceFunc(&ConstraintInput{
					StartInput: nm.seedPos,
					EndInput:   k.Q(),
				})
				nm.nnLock.RUnlock()
				if dist < bestDist {
					bestDist = dist
					best = k
				}
			}
		default:
			nm.nnLock.RLock()
			if nm.ready {
				nm.nnLock.RUnlock()
				nm.neighbors <- &neighbor{bestDist, best}
				return
			}
			nm.nnLock.RUnlock()
		}
	}
}
