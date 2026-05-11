# pkg Distribution Notes

Use this path when the audience should install `box-link` by double-clicking a macOS installer.

## Suggested flow

1. Run `just package-pkg`
2. Or call `./packaging/build-pkg.sh --version <value>`
3. Optionally sign and notarize before publishing

## Example

```bash
./packaging/build-pkg.sh \
  --version v0.1.0 \
  --identifier com.example.box-link
```

## Recommended follow-up

- add Developer ID signing
- notarize with Apple before broad distribution
- attach the `.pkg` to the same GitHub Release as the tarballs
