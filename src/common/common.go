package common

type SkeletonDeployment struct {
	Machines struct {
		Provider string
		Ip       []string
	}

	Containers map[string]struct {
		Quantity    int
		Mode        string
		Granularity string
	}
}
