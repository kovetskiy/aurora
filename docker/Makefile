CACHE = 1

local:
	docker build -t aurora . --build-arg CACHE=$(CACHE)

remote:
	$(if $(HOST),,$(error HOST is not set, do not forget about ssh:// scheme))
	docker -H $(HOST) build -t aurora . --build-arg CACHE=$(CACHE)

push:
	$(if $(TAG),,$(error TAG is not set))
	docker tag aurora $(TAG)
	docker push $(TAG)
