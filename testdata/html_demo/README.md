# HTML Reporter - Before vs After

## BEFORE (Original Implementation)
- Empty line content (just line numbers and hit counts)
- No source code display
- Generic card-based layout
- No syntax highlighting
- Limited visual feedback

Example output:
```
Line 1: 5×  [empty]
Line 2: 3×  [empty]  
Line 3: 0×  [empty]
```

## AFTER (Enhanced Implementation)
- Full source code display
- SQL syntax highlighting
- Go-style coverage layout
- Color-coded coverage indicators
- Interactive file navigation

Example output:
```
1    5    CREATE TABLE users (
2    3        id SERIAL PRIMARY KEY,
3    0        username VARCHAR(50)
```

With colors:
- Line 1-2: Green background (covered)
- Line 3: Red background (uncovered)
- Keywords: Blue
- Strings: Red
- Comments: Green

## Key Features

### Visual Style
✓ Matches \go tool cover -html\ look and feel
✓ Top bar with file selector dropdown
✓ Summary bar with total coverage percentage
✓ File navigation sidebar with clickable links
✓ Monospace font for code display

### Coverage Display
✓ Green (#c0ffc0) for covered lines (cov1-cov8)
✓ Red (#ffcccc) for uncovered lines (cov0)
✓ Gray (#f5f5f5) for non-tracked lines
✓ Hit count displayed next to each tracked line

### SQL Syntax Highlighting
✓ 60+ SQL keywords recognized
✓ Common functions (COUNT, SUM, NOW, etc.)
✓ String literals in single quotes
✓ Line comments (--)
✓ Numeric literals

### Technical Implementation
- Reads actual source files from disk
- Regex-based syntax highlighting
- HTML entity escaping for safety
- Responsive design
- No external dependencies

## Demo

Open \	estdata/html_demo/sample_report.html\ in any browser to see the result.

The report includes:
- Sample SQL file with various statements
- Mixed coverage (82.35% total)
- Covered lines (green)
- Uncovered lines (red - IF branch not taken)
- Comments and blank lines (gray - not tracked)

