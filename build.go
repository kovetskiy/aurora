package main

type build struct {
	database *database
	pkg      pkg
}

type container struct {
	Name string
}

func (build *build) Process() {
	infof("building %s", build.pkg.Name)

	container
}
