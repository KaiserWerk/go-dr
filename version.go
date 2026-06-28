package godr

import "time"

// Contains reports whether ts falls into the version validity window.
func (v Version) Contains(ts time.Time) bool {
	if ts.Before(v.ValidFrom) {
		return false
	}
	if v.ValidTo != nil && ts.After(*v.ValidTo) {
		return false
	}
	return true
}

// VersionAt returns the first version window containing ts.
func (d *LegalDocument) VersionAt(ts time.Time) *Version {
	if d == nil || len(d.Versions) == 0 {
		return nil
	}
	for i := range d.Versions {
		if d.Versions[i].Contains(ts) {
			v := d.Versions[i]
			if v.ValidTo != nil {
				cpy := *v.ValidTo
				v.ValidTo = &cpy
			}
			return &v
		}
	}
	return nil
}
