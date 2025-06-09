# Cost Guidelines for Claude Code Operations

## Cost Estimation Framework

### Token Usage Categories

#### Low Cost (<1K tokens)
- Single file reads under 100 lines
- Simple bash commands (ls, pwd, git status)
- Basic grep searches with specific patterns
- Individual file edits

**Typical operations:**
```bash
# ~50-200 tokens each
git status
ls /specific/path
grep "specific_function" file.go
```

#### Medium Cost (1K-5K tokens)
- Multiple related file reads
- Complex searches across file types
- Build operations with output
- Directory exploration of specific components

**Typical operations:**
```bash
# ~1K-3K tokens each
Multiple file reads for feature implementation
Targeted grep across component directory
Build verification with error analysis
```

#### High Cost (5K-15K tokens)
- Broad codebase exploration
- Complex multi-component analysis
- Large file processing
- Comprehensive debugging across services

**Requires user confirmation with alternatives**

#### Very High Cost (>15K tokens)
- Full codebase investigation
- Extensive refactoring analysis
- Complete system architecture review
- Large-scale debugging sessions

**Requires explicit user approval and justification**

## Cost Reduction Techniques

### 1. Targeted Searches
```
Instead of: "Search entire codebase for authentication"
Use: "Grep for 'auth' in web/src/grpc/ and go/grpc/"
Savings: ~10K tokens → ~2K tokens
```

### 2. Incremental Approach
```
Instead of: "Analyze all components for performance issues"
Use: "Start with go/cmd/ performance, then expand if needed"
Savings: ~15K tokens → ~3K tokens per step
```

### 3. Specific File Targeting
```
Instead of: "Read all Vue components"
Use: "Read components related to session management"
Savings: ~8K tokens → ~2K tokens
```

### 4. Context-Aware Operations
```
Instead of: "Full docker-compose analysis"
Use: "Check specific service configuration in docker-compose.yml"
Savings: ~5K tokens → ~1K tokens
```

## User Confirmation Templates

### High Cost Warning
```
⚠️ High Cost Operation Detected
Operation: [Description]
Estimated duration: X-Y minutes
Estimated cost: ~X,XXX tokens

This operation may consume significant resources.

Proceed? (y/n)

Lower-cost alternatives:
1. Targeted approach: [Description] (~X tokens)
2. Incremental method: [Description] (~X tokens)
3. Focused analysis: [Description] (~X tokens)

Which approach would you prefer? (1/2/3/proceed)
```

### Very High Cost Warning
```
🚨 Very High Cost Operation
Operation: [Description]
Estimated duration: X+ minutes
Estimated cost: ~XX,XXX tokens

This is a resource-intensive operation that may impact your usage limits.

Justification required. This operation is necessary because:
[ User must explain why simpler approaches won't work ]

Recommended alternatives:
1. [Specific lower-cost approach]
2. [Alternative methodology]
3. [Phased implementation]

Continue only if alternatives are insufficient.
```

## Optimization Strategies by Operation Type

### Code Analysis
- Use AST parsing hints instead of full file reads
- Target specific functions/classes instead of entire files
- Use symbol searches instead of text searches

### Debugging
- Start with error logs and stack traces
- Focus on recently changed files first
- Use incremental investigation approach

### Feature Implementation
- Examine similar existing features first
- Read component interfaces before implementations
- Use targeted searches for patterns and conventions

### Build Issues
- Check build scripts and configs first
- Use incremental builds to isolate problems
- Focus on error messages rather than full output analysis

## Cost Monitoring

### Session Tracking
Keep running total of token usage per session:
- Low: 0-5K tokens (normal operation)
- Medium: 5K-15K tokens (complex task)
- High: 15K+ tokens (requires justification)

### Operation Logging
For each major operation, log:
- Estimated vs actual token usage
- Duration taken
- User satisfaction with cost/benefit ratio
- Opportunities for optimization

### Feedback Loop
Continuously improve estimates based on:
- Actual operation costs
- User preferences for cost vs speed
- Success rates of alternative approaches