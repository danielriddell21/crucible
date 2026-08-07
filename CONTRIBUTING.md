# Contributing

Pull requests are welcome.

## Prerequisites

Go 1.26 or later.

Ebiten's Linux build needs the usual X11/GL development headers:

```sh
sudo apt-get install -y libgl1-mesa-dev libxcursor-dev libxi-dev \
  libxinerama-dev libxrandr-dev libxxf86vm-dev libasound2-dev pkg-config
```

## Running tests

```sh
go test ./...
```

The `menu` and `camera` packages import Ebiten, which needs a display. On a
headless machine run the suite under a virtual one:

```sh
xvfb-run -a go test ./...
```

## Adding an engine package

1. Create `<name>/<name>.go` with a package doc comment explaining which
   family repos the code was consolidated from and what stays app-side.
2. Keep app-defined vocabularies out: if the package carries app values,
   make it generic over them.
3. Write `<name>/<name>_test.go` in package `<name>_test`. Cover the happy
   path, the boundary cases, and determinism where a seed is involved.
4. Document every exported symbol — the revive `exported` rule enforces it.
5. Add a row to the [Provenance](https://github.com/danielriddell21/crucible/wiki/Provenance) wiki page recording where
   the package came from and what stays app-side.
