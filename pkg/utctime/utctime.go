// package utctime implements a Now function which saves writing time.Now().UTC() everywhere
package utctime

import "time"

// Now returns time.Now() in the UTC time zone
func Now() time.Time {
	return time.Now().UTC()
}
