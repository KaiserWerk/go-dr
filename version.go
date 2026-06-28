package godr

import (
	"sort"
	"time"
)

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

// EnsureVersions normalizes existing versions and derives a fallback window if none is set.
func (d *LegalDocument) EnsureVersions() {
	if d == nil {
		return
	}

	if len(d.Versions) == 0 {
		if v, ok := d.derivedVersionWindow(); ok {
			d.Versions = []Version{v}
		}
		return
	}

	normalized := normalizeVersions(d.Versions)
	if len(normalized) == 0 {
		d.Versions = nil
		if v, ok := d.derivedVersionWindow(); ok {
			d.Versions = []Version{v}
		}
		return
	}

	d.Versions = normalized
	if d.EffectiveFrom == nil && !normalized[0].ValidFrom.IsZero() {
		vf := normalized[0].ValidFrom
		d.EffectiveFrom = &vf
	}
	last := normalized[len(normalized)-1]
	if d.EffectiveTo == nil && last.ValidTo != nil {
		vt := *last.ValidTo
		d.EffectiveTo = &vt
	}
}

func (d *LegalDocument) derivedVersionWindow() (Version, bool) {
	if d == nil {
		return Version{}, false
	}

	v := Version{}
	if d.EffectiveFrom != nil {
		v.ValidFrom = *d.EffectiveFrom
	} else if d.PublishedAt != nil {
		v.ValidFrom = *d.PublishedAt
	}
	if d.EffectiveTo != nil {
		vt := *d.EffectiveTo
		v.ValidTo = &vt
	}

	if v.ValidFrom.IsZero() && v.ValidTo == nil {
		return Version{}, false
	}
	if v.ValidTo != nil && !v.ValidFrom.IsZero() && v.ValidTo.Before(v.ValidFrom) {
		return Version{}, false
	}
	return v, true
}

func normalizeVersions(in []Version) []Version {
	if len(in) == 0 {
		return nil
	}

	out := make([]Version, 0, len(in))
	for _, v := range in {
		if v.ValidTo != nil && !v.ValidFrom.IsZero() && v.ValidTo.Before(v.ValidFrom) {
			continue
		}
		copyV := v
		if v.ValidTo != nil {
			vt := *v.ValidTo
			copyV.ValidTo = &vt
		}
		out = append(out, copyV)
	}
	if len(out) == 0 {
		return nil
	}

	sort.SliceStable(out, func(i, j int) bool {
		a := out[i].ValidFrom
		b := out[j].ValidFrom
		if a.Equal(b) {
			if out[i].ValidTo == nil {
				return false
			}
			if out[j].ValidTo == nil {
				return true
			}
			return out[i].ValidTo.Before(*out[j].ValidTo)
		}
		if a.IsZero() {
			return true
		}
		if b.IsZero() {
			return false
		}
		return a.Before(b)
	})

	compacted := make([]Version, 0, len(out))
	for _, v := range out {
		if len(compacted) == 0 {
			compacted = append(compacted, v)
			continue
		}
		prev := compacted[len(compacted)-1]
		if prev.ValidFrom.Equal(v.ValidFrom) && sameTimePtr(prev.ValidTo, v.ValidTo) {
			continue
		}
		compacted = append(compacted, v)
	}

	return compacted
}

func sameTimePtr(a, b *time.Time) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	return a.Equal(*b)
}
