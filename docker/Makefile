build:
	$(if $(HOST),,$(error HOST is not set, do not forget about ssh:// scheme))
	docker -H $(HOST) build -t aurora .
