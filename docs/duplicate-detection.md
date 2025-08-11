# Duplicate Code Detection

HTNN now includes automated duplicate code detection to help maintain code quality and reduce maintenance burden.

## Overview

The duplicate detection system identifies similar code blocks across the codebase that could potentially be refactored to improve maintainability.

## Tools Used

1. **dupl linter** - Detects duplicate code blocks in Go files
2. **dupword linter** - Detects duplicate words in comments and strings
3. **Custom analysis script** - Provides comprehensive duplicate analysis across all modules

## Running Duplicate Detection

### Via Make Target
```bash
# Run duplicate detection as part of the full lint suite
make lint

# Run only duplicate detection
make lint-duplicates
```

### Via Script Directly
```bash
# Default threshold (50 tokens)
./scripts/detect-duplicates.sh

# Custom threshold (100 tokens)
./scripts/detect-duplicates.sh 100
```

### Via golangci-lint
The dupl and dupword linters are now enabled in `.golangci.yml` and will run as part of the normal linting process.

## Configuration

### Thresholds
- **dupl**: 100 tokens (configured in `.golangci.yml`)
- **detect-duplicates.sh**: 50 tokens default, configurable via command line

### Exclusions
The following files are excluded from duplicate detection:
- `.pb.go` files (generated protobuf code)
- `.pb.validate.go` files (generated validation code)

## Common Duplicate Patterns Found

Based on the analysis, common patterns include:

1. **Test Setup/Teardown Code**
   - Similar test initialization patterns
   - Repeated assertion blocks
   - Common test data structures

2. **Plugin Configuration**
   - Similar plugin validation logic
   - Repeated configuration patterns
   - Common error handling

3. **Generated Code**
   - Protobuf validation functions
   - Similar validation patterns across different types

## Recommendations for Refactoring

1. **Extract Test Helpers**
   ```go
   // Instead of duplicating test setup in multiple files
   func setupTestEnvironment(t *testing.T) *TestEnv {
       // Common setup logic
   }
   ```

2. **Create Base Test Structures**
   ```go
   // Common test case structure for plugin tests
   type PluginTestCase struct {
       Name   string
       Config interface{}
       Expect func(*testing.T, *http.Response)
   }
   ```

3. **Abstract Common Patterns**
   ```go
   // Extract common validation logic
   func validatePluginConfig(config interface{}) error {
       // Common validation patterns
   }
   ```

## CI Integration

Duplicate detection runs as part of the CI lint process. If duplicates above the threshold are found, the build will fail with suggestions for refactoring.

## Threshold Guidelines

- **50 tokens**: Catches most meaningful duplicates
- **100 tokens**: More conservative, catches only significant duplicates
- **Lower thresholds**: May catch trivial duplicates that aren't worth refactoring

Adjust thresholds based on your team's tolerance for duplication vs. refactoring effort.