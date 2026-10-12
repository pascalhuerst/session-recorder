package render

import (
	"math/bits"

	"github.com/mewkiz/flac/frame"
)

// The FLAC library only writes what it is told to: it turns a subframe's
// prediction method, order and Rice parameters into bits, but does not choose
// them. This file does the choosing, per block:
//
//   - which stereo representation (left/right, left/side, side/right or
//     mid/side) is cheapest,
//   - per channel of that representation, which fixed predictor (order 0-4)
//     leaves the smallest residual,
//   - how to partition that residual and which Rice parameter each partition
//     gets.
//
// Fixed predictors only - no LPC analysis - which is roughly what `flac -0`
// to `-2` do, and already gets most of the way to the usual compression.

const (
	// Highest fixed predictor order FLAC defines.
	maxFixedOrder = 4
	// Residual partition order limit: at most 2^6 partitions per block. The
	// streamable subset allows 8; beyond 6 the gain is negligible.
	maxPartOrder = 6
	// Largest Rice parameter of each coding method (the all-ones value is the
	// escape code, which we never emit).
	maxRice1Param = 14
	maxRice2Param = 30

	// Bits of a subframe header without wasted-bits info: 1 padding bit, 6 bits
	// of type, 1 wasted-bits flag.
	subframeHeaderBits = 8
	// Residual coding method (2 bits) and partition order (4 bits).
	residualHeaderBits = 6
)

// subframePlan is how one channel of a block is going to be stored, and what
// that costs.
type subframePlan struct {
	pred   frame.Pred
	order  int
	method frame.ResidualCodingMethod
	// Rice parameter per partition; len(params) == 1<<partOrder.
	partOrder int
	params    []uint
	// Exact size of the encoded subframe in bits.
	bits uint64
}

// apply copies the plan into the subframe header the encoder reads.
func (p *subframePlan) apply(sub *frame.Subframe) {
	sub.SubHeader = frame.SubHeader{Pred: p.pred, Order: p.order}
	if p.pred != frame.PredFixed {
		return
	}
	partitions := make([]frame.RicePartition, len(p.params))
	for i, k := range p.params {
		partitions[i].Param = k
	}
	sub.ResidualCodingMethod = p.method
	sub.RiceSubframe = &frame.RiceSubframe{PartOrder: p.partOrder, Partitions: partitions}
}

// predictor holds the scratch buffers for planning subframes, so a multi-hour
// stream does not allocate per block.
type predictor struct {
	// residual[o] is the order-o fixed-predictor residual of the current
	// signal; entries below index o are unused.
	residual [maxFixedOrder + 1][]int32
	// folded is the zigzag-folded residual of the chosen order.
	folded []uint32
	// sums[j] is the sum of folded residuals in partition j at the partition
	// order currently being evaluated.
	sums []uint64
}

func newPredictor(blockSize int) *predictor {
	p := &predictor{
		folded: make([]uint32, blockSize),
		sums:   make([]uint64, 1<<maxPartOrder),
	}
	for o := range p.residual {
		p.residual[o] = make([]int32, blockSize)
	}
	return p
}

// foldResidual maps a signed residual to the unsigned value Rice coding
// stores, exactly as the encoder does.
func foldResidual(r int32) uint32 {
	if r < 0 {
		return uint32(-r)<<1 - 1
	}
	return uint32(r) << 1
}

// riceParam returns the Rice parameter (at most maxParam) that minimizes the
// estimated size of a partition of n residuals whose folded values sum to sum,
// and that estimate in bits.
//
// A residual u costs (u>>k)+1+k bits. Summed over the partition, sum>>k is a
// close stand-in for the first term that needs no second pass over the data.
func riceParam(sum uint64, n int, maxParam uint) (uint, uint64) {
	var k uint
	if n > 0 && sum > uint64(n) {
		// The estimate is minimal around k = log2(mean residual).
		k = uint(bits.Len64(sum/uint64(n))) - 1
	}
	best, bestBits := uint(0), ^uint64(0)
	lo := uint(0)
	if k > 0 {
		lo = k - 1
	}
	for c := lo; c <= k+1 && c <= maxParam; c++ {
		estimate := uint64(n)*uint64(c+1) + sum>>c
		if estimate < bestBits {
			best, bestBits = c, estimate
		}
	}
	if bestBits == ^uint64(0) {
		// Even the smallest candidate exceeded maxParam: clamp.
		return maxParam, uint64(n)*uint64(maxParam+1) + sum>>maxParam
	}
	return best, bestBits
}

// plan decides how to store samples (one channel of one block, already in the
// stereo representation it will be written in) at bps bits per sample.
func (p *predictor) plan(samples []int32, bps uint) subframePlan {
	n := len(samples)
	verbatim := subframePlan{pred: frame.PredVerbatim, bits: subframeHeaderBits + uint64(n)*uint64(bps)}
	if n == 0 {
		return verbatim
	}

	constant := true
	for _, s := range samples[1:] {
		if s != samples[0] {
			constant = false
			break
		}
	}
	if constant {
		return subframePlan{pred: frame.PredConstant, bits: subframeHeaderBits + uint64(bps)}
	}

	// Residuals of every fixed order by repeated differencing; keep the order
	// whose residual is smallest in total magnitude.
	copy(p.residual[0][:n], samples)
	maxOrder := min(maxFixedOrder, n-1)
	bestOrder, bestMagnitude := 0, ^uint64(0)
	for o := 0; o <= maxOrder; o++ {
		res := p.residual[o][:n]
		if o > 0 {
			prev := p.residual[o-1][:n]
			for i := o; i < n; i++ {
				res[i] = prev[i] - prev[i-1]
			}
		}
		var magnitude uint64
		for _, r := range res[o:] {
			if r < 0 {
				r = -r
			}
			magnitude += uint64(r)
		}
		if magnitude < bestMagnitude {
			bestOrder, bestMagnitude = o, magnitude
		}
	}

	order := bestOrder
	folded := p.folded[:n]
	for i := order; i < n; i++ {
		folded[i] = foldResidual(p.residual[order][i])
	}

	// The finest partitioning the block allows: every partition must have the
	// same length, and the first one must still hold residuals after the
	// warm-up samples.
	finest := 0
	for finest < maxPartOrder && n%(1<<(finest+1)) == 0 && n>>(finest+1) > order {
		finest++
	}

	best := subframePlan{bits: ^uint64(0)}
	for _, coding := range []struct {
		method    frame.ResidualCodingMethod
		maxParam  uint
		paramBits uint64
	}{
		{frame.ResidualCodingMethodRice1, maxRice1Param, 4},
		{frame.ResidualCodingMethodRice2, maxRice2Param, 5},
	} {
		// Partition sums at the finest order; coarser orders merge pairs.
		parts := 1 << finest
		size := n >> finest
		sums := p.sums[:parts]
		for j := range sums {
			start := j * size
			if j == 0 {
				start = order
			}
			var sum uint64
			for _, u := range folded[start : (j+1)*size] {
				sum += uint64(u)
			}
			sums[j] = sum
		}

		bestPartOrder, bestEstimate := 0, ^uint64(0)
		for po := finest; ; po-- {
			parts = 1 << po
			size = n >> po
			var estimate uint64
			for j := 0; j < parts; j++ {
				count := size
				if j == 0 {
					count -= order
				}
				_, partBits := riceParam(sums[j], count, coding.maxParam)
				estimate += coding.paramBits + partBits
			}
			if estimate < bestEstimate {
				bestPartOrder, bestEstimate = po, estimate
			}
			if po == 0 {
				break
			}
			for j := 0; j < parts/2; j++ {
				sums[j] = sums[2*j] + sums[2*j+1]
			}
		}

		// Exact size at the chosen partition order.
		parts = 1 << bestPartOrder
		size = n >> bestPartOrder
		params := make([]uint, parts)
		total := subframeHeaderBits + uint64(order)*uint64(bps) + residualHeaderBits + uint64(parts)*coding.paramBits
		for j := 0; j < parts; j++ {
			start := j * size
			if j == 0 {
				start = order
			}
			part := folded[start : (j+1)*size]
			var sum uint64
			for _, u := range part {
				sum += uint64(u)
			}
			k, _ := riceParam(sum, len(part), coding.maxParam)
			params[j] = k
			for _, u := range part {
				total += uint64(u>>k) + uint64(k) + 1
			}
		}
		if total < best.bits {
			best = subframePlan{
				pred:      frame.PredFixed,
				order:     order,
				method:    coding.method,
				partOrder: bestPartOrder,
				params:    params,
				bits:      total,
			}
		}
		// The 5-bit parameter only pays off when the 4-bit one ran into its
		// limit somewhere - for 16-bit audio, practically never.
		saturated := false
		for _, k := range params {
			if k == coding.maxParam {
				saturated = true
				break
			}
		}
		if !saturated {
			break
		}
	}

	if best.bits >= verbatim.bits {
		return verbatim
	}
	return best
}

// stereoPlan is the cheapest way to store one stereo block: the channel
// assignment and the plan for each of its two subframes.
type stereoPlan struct {
	channels  frame.Channels
	subframes [2]subframePlan
}

// planStereo compares the four stereo representations FLAC offers for a
// block and returns the cheapest. mid and side are scratch buffers of at
// least len(left) samples.
//
// The encoder derives mid and side itself from the left/right samples it is
// handed (side = left - right, mid = (left + right) >> 1); the same formulas
// are used here so the plans match what actually gets encoded. The side
// channel needs one bit more per sample than the input.
func (p *predictor) planStereo(left, right, mid, side []int32, bps uint) stereoPlan {
	n := len(left)
	mid, side = mid[:n], side[:n]
	for i := range left {
		l, r := left[i], right[i]
		mid[i] = int32((int64(l) + int64(r)) >> 1)
		side[i] = l - r
	}

	planLeft := p.plan(left, bps)
	planRight := p.plan(right, bps)
	planMid := p.plan(mid, bps)
	planSide := p.plan(side, bps+1)

	best := stereoPlan{channels: frame.ChannelsLR, subframes: [2]subframePlan{planLeft, planRight}}
	bestBits := planLeft.bits + planRight.bits
	for _, candidate := range []stereoPlan{
		{frame.ChannelsLeftSide, [2]subframePlan{planLeft, planSide}},
		{frame.ChannelsSideRight, [2]subframePlan{planSide, planRight}},
		{frame.ChannelsMidSide, [2]subframePlan{planMid, planSide}},
	} {
		if candidateBits := candidate.subframes[0].bits + candidate.subframes[1].bits; candidateBits < bestBits {
			best, bestBits = candidate, candidateBits
		}
	}
	return best
}
