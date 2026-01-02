# Convert a CSV export to SBR data

Converts iMazing's CSV call history export format to SBR data produced by SMS Backup and Restore Pro for Android.

## Usage

```bash
iphone2sbr [options]
```

## Options

- `-log-level` (int, default: 2)
  Log level for the application output:
  - `0` = warn
  - `1` = info
  - `2` = debug

- `-import-file` (string, default: "")
  Path to the CSV file to import (iMazing call history export)

- `-collection-file` (string, default: "")
  Path to the collection file to append converted calls to

- `-tag` (string, default: "")
  Tag to apply to all imported calls (currently unused)

All options can also be set via environment variables with the prefix `IPHONE2SBR_`, for example:
- `IPHONE2SBR_LOG_LEVEL`
- `IPHONE2SBR_IMPORT_FILE`
- `IPHONE2SBR_COLLECTION_FILE`
- `IPHONE2SBR_TAG`