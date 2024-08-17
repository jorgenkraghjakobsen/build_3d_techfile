
run:
	go run build_3d_techfile.go

build:
	go build -o sg13g2 build_3d_techfile.go


install:
	cp sg13g2.txt $(HOME)/opentools/GDS3D/techfiles/
