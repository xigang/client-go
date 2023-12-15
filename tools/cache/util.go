package cache

import (
	"encoding/json"
	"sort"
)

type MultiClusterResourceVersion struct {
	Rvs    map[string]string
	IsZero bool
}

func NewMultiClusterResourceVersionWithCapacity(capacity int) *MultiClusterResourceVersion {
	return &MultiClusterResourceVersion{
		Rvs: make(map[string]string, capacity),
	}
}

// `s` format: `{"cluster1":"123","cluster2":"456"}`
func NewMultiClusterResourceVersionFromString(s string) *MultiClusterResourceVersion {
	m := &MultiClusterResourceVersion{
		Rvs: make(map[string]string),
	}
	if s == "" {
		return m
	}
	if s == "0" {
		m.IsZero = true
		return m
	}

	// if invalid, ignore the version
	_ = json.Unmarshal([]byte(s), &m.Rvs)
	return m
}

func (m *MultiClusterResourceVersion) Set(cluster, rv string) {
	m.Rvs[cluster] = rv
	if rv != "0" {
		m.IsZero = false
	}
}

func (m *MultiClusterResourceVersion) Get(cluster string) string {
	if m.IsZero {
		return "0"
	}
	return m.Rvs[cluster]
}

func (m *MultiClusterResourceVersion) String() string {
	if m.IsZero {
		return "0"
	}

	if len(m.Rvs) == 0 {
		return ""
	}
	buf := marshalRvs(m.Rvs)
	return string(buf)
}

func marshalRvs(rvs map[string]string) []byte {
	// We must make sure the returned ResourceVersion string is stable, because client might use this string
	// for equality comparison. But hashmap's key iteration order is not determined.
	// So we can't use json encoding library directly.
	// Instead, we convert the map to a slice, sort the slice by `Cluster` field, then manually build a string.
	// The result string resembles json encoding result to keep compatible with older version of proxy.

	if len(rvs) == 0 {
		// need to keep sync with `func (m *multiClusterResourceVersion) String() string`
		return nil
	}

	type onWireRvs struct {
		Cluster         string
		ResourceVersion string
	}

	slice := make([]onWireRvs, 0, len(rvs))

	for clusterName, version := range rvs {
		obj := onWireRvs{clusterName, version}
		slice = append(slice, obj)
	}

	sort.Slice(slice, func(i, j int) bool {
		return slice[i].Cluster < slice[j].Cluster
	})

	var encoded = make([]byte, 0, (len(slice[0].Cluster)+len(slice[0].ResourceVersion)+20)*len(slice)+2)
	encoded = append(encoded, '{')
	for i, n := 0, len(slice); i < n; i++ {
		encoded = append(encoded, '"')
		encoded = append(encoded, slice[i].Cluster...)
		encoded = append(encoded, `":"`...)
		encoded = append(encoded, slice[i].ResourceVersion...)
		encoded = append(encoded, '"')

		if i != n-1 {
			encoded = append(encoded, ',')
		}
	}
	encoded = append(encoded, '}')

	return encoded
}