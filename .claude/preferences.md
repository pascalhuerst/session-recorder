# Claude Code Preferences

Configuration preferences for optimal Claude Code performance with this session recorder project.

## Tool Usage Preferences

### Search Strategy
```
1. Grep/Glob for specific patterns and file types
2. Task tool for complex multi-step searches  
3. Read specific files when paths are known
4. Directory exploration as last resort
```

### Operation Batching
```
Batch these operations in single messages:
- git status + git diff + git log
- Multiple file reads for related components
- Build + test + lint operations
- Docker compose up + logs + status checks
```

### Timeout Configuration
```yaml
timeouts:
  quick_commands: 30000    # ls, git status, pwd
  build_operations: 300000 # go build, npm install, make
  test_suites: 600000      # comprehensive test runs
  network_ops: 180000      # docker pull, git clone
  file_operations: 60000   # large file reads, searches
```

## Project-Specific Optimizations

### Component Mapping
```
Audio capture issues -> cpp/chunk-sink-client/
Backend API problems -> go/cmd/, go/grpc/
Web interface bugs -> web/src/
Protocol changes -> protocols/proto/, protocols/
Build problems -> build.sh, docker-build.sh
```

### Common Workflows
```
1. Feature development:
   - Search relevant component first
   - Read related files in parallel
   - Implement changes with batched verification

2. Bug investigation:
   - Use Task tool for broad issue search
   - Read logs and error traces simultaneously
   - Test fixes with appropriate timeouts

3. Build troubleshooting:
   - Check build scripts and configs in parallel
   - Verify dependencies with longer timeouts
   - Run incremental builds to isolate issues
```

### Context Optimization
- Read file sections (offset/limit) for large files
- Use include patterns in Grep for targeted searches
- Avoid reading build artifacts and generated files
- Cache search results within conversation context

## Quality Assurance

### Pre-commit Checks
Always run these with appropriate timeouts:
```bash
# Use 300s timeout for build verification
./build.sh

# Use 180s timeout for docker operations  
./docker-build.sh up --build

# Use 120s timeout for protocol generation
cd protocols && make all
```

### Error Prevention
- Validate file paths before operations
- Check service dependencies before starting components
- Verify environment variables for S3/MinIO operations
- Use incremental builds to reduce timeout risks