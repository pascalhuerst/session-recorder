//go:build race

package render

// raceEnabled is true when the test binary is built with the race detector
// (`go test -race`), which also turns on the `checkptr` unsafe-pointer checks.
const raceEnabled = true
