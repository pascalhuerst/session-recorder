# Claude Code Instructions

This directory contains configuration and instructions for Claude Code when working with this session recorder project.

## Core Guidelines

### CLI Operations & Timeouts
Always use timeout parameters for CLI operations to prevent hanging:

```
- Quick operations (ls, git status, simple commands): 30s timeout
- Build operations (go build, npm install): 300s timeout  
- Long-running tests and complex builds: 600s timeout
- Network operations (docker pulls, downloads): 180s timeout
```

### Cost Optimization Strategies

#### Batch Operations
- Use single messages with multiple tool calls for parallel execution
- Example: Run `git status` and `git diff` simultaneously
- Batch file reads when examining multiple related files

#### Efficient Search Patterns
- Use targeted Grep/Glob searches instead of broad exploration
- Use Task tool for complex multi-round searches
- Read specific file sections rather than entire large files
- Prefer existing file inspection over creating new exploration files

#### Tool Selection Priority
1. **Targeted tools first**: Grep, Glob for specific patterns
2. **Task tool**: For complex searches requiring multiple rounds
3. **Direct file access**: Read specific known files
4. **Directory exploration**: Only when absolutely necessary

### Session Recorder Specific

#### Development Workflow
Always follow this sequence for development tasks:
1. Use timeouts on all CLI operations
2. Batch related operations (status checks, builds, tests)
3. Read relevant files in parallel when possible
4. Use targeted searches to understand codebase structure
5. Verify changes with appropriate test commands

#### Common Operations
- **Build verification**: Use 300s timeout for `./build-audio-client.sh`
- **Service startup**: Use 180s timeout for docker operations
- **Test execution**: Use 600s timeout for comprehensive test suites
- **Protocol generation**: Use 120s timeout for `make all` in protocols/

#### File Organization Awareness
- `protocols/`: gRPC definitions - use when working with API changes
- `go/`: Backend services - check before modifying server logic  
- `web/`: Frontend application - examine before UI/UX changes
- `cpp/`: Audio capture client - review for hardware integration

### Cost Estimation & User Confirmation

Before executing operations, provide estimates:

#### Duration Estimates
- **Quick operations**: 5-30 seconds (git status, ls, simple reads)
- **Medium operations**: 1-5 minutes (builds, installs, complex searches)
- **Long operations**: 5-10 minutes (full test suites, large builds)

#### Cost Estimates (approximate token usage)
- **Low cost**: <1K tokens (single file reads, simple commands)
- **Medium cost**: 1K-5K tokens (multiple file operations, targeted searches)
- **High cost**: 5K-15K tokens (broad codebase exploration, complex analysis)
- **Very high cost**: >15K tokens (extensive multi-component investigation)

#### User Confirmation Required
For operations estimated at **high cost or higher** (>5K tokens):

```
⚠️ High Cost Operation Detected
Estimated duration: X minutes
Estimated cost: ~X tokens

Confirm execution? (y/n)

Lower-cost alternatives:
1. [Specific targeted approach with ~X tokens]
2. [Alternative method with ~X tokens] 
3. [Minimal viable approach with ~X tokens]
```

#### Cost Reduction Strategies
When offering alternatives:
- **Targeted search** instead of broad exploration
- **Specific file reads** instead of directory traversal
- **Incremental approach** instead of comprehensive analysis
- **Task tool delegation** for complex multi-step operations

### Error Handling
- Always include timeout parameters to prevent indefinite waits
- Use appropriate timeout values based on operation complexity
- Prefer graceful degradation over operation failure
- Document any timeout-related issues for future optimization

### Performance Guidelines
- Minimize context usage by reading only necessary file sections
- Use parallel tool execution for independent operations
- Avoid redundant file reads within the same session
- Cache expensive operations results when possible