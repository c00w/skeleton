package common

type SkeletonDeployment struct {
	Machines struct {
		Provider string
		Ip       []string
	}

	Containers map[string]struct {
		Source      string
		Quantity    int
		Mode        string
		Granularity string
		Expose      []string
	}
}
