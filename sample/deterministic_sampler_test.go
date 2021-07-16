package sample

import (
	"math/rand"
	"testing"
)

const requestIDBytes = `abcdef0123456789`

func init() {
	rand.Seed(1)
}

func randomRequestID() string {
	// create request ID roughly resembling something you would get from
	// AWS ALB, e.g.,
	//
	// 1-5ababc0a-4df707925c1681932ea22a20
	//
	// The AWS docs say the middle bit is "time in seconds since epoch",
	// (implying base 10) but the above represents an actual Root= ID from
	// an ALB access log, so... yeah.
	reqID := "1-"
	for i := 0; i < 8; i++ {
		reqID += string(requestIDBytes[rand.Intn(len(requestIDBytes))])
	}
	reqID += "-"
	for i := 0; i < 24; i++ {
		reqID += string(requestIDBytes[rand.Intn(len(requestIDBytes))])
	}
	return reqID
}

func assertEqual(t *testing.T, a interface{}, b interface{}) {
	if a != b {
		t.Fatalf("%v != %v", a, b)
	}
}

func TestDeterministicSamplerDatapoints(t *testing.T) {
	s, _ := NewDeterministicSampler(17)
	a := s.Sample("hello")
	assertEqual(t, a, false)
	a = s.Sample("hello")
	assertEqual(t, a, false)
	a = s.Sample("world")
	assertEqual(t, a, false)
	a = s.Sample("this5")
	assertEqual(t, a, true)
}

func TestDeterministicSampler(t *testing.T) {
	const (
		nRequestIDs             = 200000
		acceptableMarginOfError = 0.05
	)

	testSampleRates := []uint{1, 2, 10, 50, 100}

	// distribution for sampling should be good
	for _, sampleRate := range testSampleRates {
		ds, err := NewDeterministicSampler(sampleRate)
		if err != nil {
			t.Fatalf("error creating deterministic sampler: %s", err)
		}

		nSampled := 0

		for i := 0; i < nRequestIDs; i++ {
			sampled := ds.Sample(randomRequestID())
			if sampled {
				nSampled++
			}
		}

		expectedNSampled := (nRequestIDs * (1 / float64(sampleRate)))

		// Sampling should be balanced across all request IDs
		// regardless of sample rate. If we cross this threshold, flunk
		// the test.
		unacceptableLowBound := int(expectedNSampled - (expectedNSampled * acceptableMarginOfError))
		unacceptableHighBound := int(expectedNSampled + (expectedNSampled * acceptableMarginOfError))
		if nSampled < unacceptableLowBound || nSampled > unacceptableHighBound {
			t.Fatal("Sampled more or less than we should have: ", nSampled, "(sample rate ", sampleRate, ")")
		}
	}

	s1, _ := NewDeterministicSampler(2)
	s2, _ := NewDeterministicSampler(2)
	sampleString := "#hashbrowns"
	firstAnswer := s1.Sample(sampleString)

	// sampler should not give different answers for subsequent runs
	for i := 0; i < 25; i++ {
		s1Answer := s1.Sample(sampleString)
		s2Answer := s2.Sample(sampleString)
		if s1Answer != firstAnswer || s2Answer != firstAnswer {
			t.Fatalf("deterministic samplers were not deterministic:\n\titeration: %d\n\ts1Answer was %t\n\ts2Answer was %t\n\tfirstAnswer was %t", i, s1Answer, s2Answer, firstAnswer)
		}
	}
}

// Tests the deterministic sampler against a specific set of determiniants for specific results,
// which should be consistent across beelines
func TestDeterministicBeelineInterop(t *testing.T) {
	ds, err := NewDeterministicSampler(2)
	if err != nil {
		t.Fatalf("error creating deterministic sampler: %s", err)
	}

	ids := []string{
		"4YeYygWjTZ41zOBKUoYUaSVxPGm78rdU",
		"iow4KAFBl9u6lF4EYIcsFz60rXGvu7ph",
		"EgQMHtruEfqaqQqRs5nwaDXsegFGmB5n",
		"UnVVepVdyGIiwkHwofyva349tVu8QSDn",
		"rWuxi2uZmBEprBBpxLLFcKtXHA8bQkvJ",
		"8PV5LN1IGm5T0ZVIaakb218NvTEABNZz",
		"EMSmscnxwfrkKd1s3hOJ9bL4zqT1uud5",
		"YiLx0WGJrQAge2cVoAcCscDDVidbH4uE",
		"IjD0JHdQdDTwKusrbuiRO4NlFzbPotvg",
		"ADwiQogJGOS4X8dfIcidcfdT9fY2WpHC",
		"DyGaS7rfQsMX0E6TD9yORqx7kJgUYvNR",
		"MjOCkn11liCYZspTAhdULMEfWJGMHvpK",
		"wtGa41YcFMR5CBNr79lTfRAFi6Vhr6UF",
		"3AsMjnpTBawWv2AAPDxLjdxx4QYl9XXb",
		"sa2uMVNPiZLK52zzxlakCUXLaRNXddBz",
		"NYH9lkdbvXsiUFKwJtjSkQ1RzpHwWloK",
		"8AwzQeY5cudY8YUhwxm3UEP7Oos61RTY",
		"ADKWL3p5gloRYO3ptarTCbWUHo5JZi3j",
		"UAnMARj5x7hkh9kwBiNRfs5aYDsbHKpw",
		"Aes1rgTLMNnlCkb9s6bH7iT5CbZTdxUw",
		"eh1LYTOfgISrZ54B7JbldEpvqVur57tv",
		"u5A1wEYax1kD9HBeIjwyNAoubDreCsZ6",
		"mv70SFwpAOHRZt4dmuw5n2lAsM1lOrcx",
		"i4nIu0VZMuh5hLrUm9w2kqNxcfYY7Y3a",
		"UqfewK2qFZqfJ619RKkRiZeYtO21ngX1",
	}
	expected := []bool{
		false,
		true,
		true,
		true,
		true,
		false,
		true,
		true,
		false,
		false,
		true,
		false,
		true,
		false,
		false,
		false,
		false,
		false,
		true,
		true,
		false,
		false,
		true,
		true,
		false,
	}

	for i := range ids {
		keep := ds.Sample(ids[i])
		if keep != expected[i] {
			t.Errorf("got unexpected deterministic sampler decision for %s: %t, expected %t", ids[i], keep, expected[i])
		}
	}
}
