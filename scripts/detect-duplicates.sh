#!/bin/bash
# Copyright The HTNN Authors.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(cd "${SCRIPT_DIR}/.." && pwd)"
LOCALBIN="${ROOT_DIR}/bin"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
THRESHOLD=${1:-50}  # Default threshold of 50 tokens
VERBOSE=${VERBOSE:-false}

echo -e "${BLUE}HTNN Duplicate Code Detection${NC}"
echo -e "${BLUE}==============================${NC}"
echo "Threshold: ${THRESHOLD} tokens"
echo ""

# Ensure dupl is installed
if ! command -v "${LOCALBIN}/dupl" &> /dev/null; then
    echo -e "${YELLOW}Installing dupl tool...${NC}"
    GOBIN="${LOCALBIN}" go install github.com/golangci/dupl@latest
fi

# Modules to check (excluding site as it's mostly docs)
MODULES=(
    "api"
    "controller" 
    "plugins"
    "types"
    "e2e"
    "tools"
)

total_clone_groups=0
total_clones=0

echo -e "${BLUE}Scanning for duplicate code blocks...${NC}"
echo ""

for module in "${MODULES[@]}"; do
    if [ -d "${ROOT_DIR}/${module}" ]; then
        echo -e "${YELLOW}=== ${module} module ===${NC}"
        
        # Run dupl and capture output
        dupl_output=$("${LOCALBIN}/dupl" -threshold "${THRESHOLD}" "${ROOT_DIR}/${module}" 2>/dev/null || true)
        
        if [ -n "$dupl_output" ]; then
            echo "$dupl_output"
            
            # Count clone groups for this module
            module_clone_groups=$(echo "$dupl_output" | grep -c "found.*clones:" || true)
            module_clones=$(echo "$dupl_output" | grep "found.*clones:" | sed 's/found \([0-9]*\) clones:.*/\1/' | awk '{sum += $1} END {print sum}')
            
            total_clone_groups=$((total_clone_groups + module_clone_groups))
            total_clones=$((total_clones + module_clones))
            
            echo ""
        else
            echo -e "${GREEN}No duplicates found${NC}"
            echo ""
        fi
    fi
done

# Summary
echo -e "${BLUE}=== Summary ===${NC}"
if [ $total_clone_groups -gt 0 ]; then
    echo -e "${RED}Found ${total_clone_groups} clone groups with ${total_clones} total clones${NC}"
    echo -e "${YELLOW}Consider refactoring these duplicates to improve maintainability${NC}"
    
    # Additional analysis
    echo ""
    echo -e "${BLUE}=== Most Common Duplicate Patterns ===${NC}"
    
    # Run dupl on all modules and analyze patterns
    all_dupl_output=$("${LOCALBIN}/dupl" -threshold "${THRESHOLD}" "${ROOT_DIR}/api" "${ROOT_DIR}/controller" "${ROOT_DIR}/plugins" "${ROOT_DIR}/types" "${ROOT_DIR}/e2e" "${ROOT_DIR}/tools" 2>/dev/null || true)
    
    if [ -n "$all_dupl_output" ]; then
        # Show files with most duplicates
        echo "Files with multiple duplicates:"
        echo "$all_dupl_output" | grep -o '[^/]*\.go:[0-9]*,[0-9]*' | cut -d':' -f1 | sort | uniq -c | sort -nr | head -5 | while read count file; do
            echo "  $file: $count occurrences"
        done
        
        echo ""
        echo "Common patterns:"
        echo "  - Test files often have similar setup/teardown code"
        echo "  - Generated .pb.validate.go files contain similar validation patterns"
        echo "  - Plugin configurations may share common structures"
        echo ""
        echo -e "${YELLOW}Recommendations:${NC}"
        echo "  1. Extract common test helpers into shared utilities"
        echo "  2. Create base test structures for plugin tests"
        echo "  3. Consider abstracting common validation patterns"
        echo "  4. Use composition over duplication for similar functionality"
    fi
    
    exit 1
else
    echo -e "${GREEN}No duplicate code found above threshold of ${THRESHOLD} tokens${NC}"
    exit 0
fi