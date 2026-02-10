#!/usr/bin/env python3
"""
Documentation Link Analyzer
Comprehensive link analysis for wiki compatibility
"""

import os
import re
from pathlib import Path
from typing import Dict, List, Tuple
from collections import defaultdict

DOCS_DIR = Path("/Users/admin/Sites/nself/.wiki")
REPORT_FILE = DOCS_DIR / "LINK-AUDIT-REPORT.md"

class LinkAnalyzer:
    def __init__(self):
        self.total_files = 0
        self.total_links = 0
        self.broken_links = []
        self.external_links = []
        self.anchor_only_links = []
        self.valid_links = []
        self.file_links = defaultdict(list)

    def extract_links(self, content: str) -> List[Tuple[str, str]]:
        """Extract all markdown links [text](url)"""
        # Pattern for markdown links
        pattern = r'\[([^\]]+)\]\(([^\)]+)\)'
        return re.findall(pattern, content)

    def is_external(self, url: str) -> bool:
        """Check if URL is external"""
        return url.startswith(('http://', 'https://', 'mailto:', 'ftp://'))

    def resolve_path(self, link_url: str, source_file: Path) -> Path:
        """Resolve relative path to absolute"""
        # Remove anchor
        clean_url = link_url.split('#')[0]
        if not clean_url:
            return None

        # Handle absolute paths from wiki root
        if clean_url.startswith('/'):
            clean_url = clean_url.lstrip('/')
            if clean_url.startswith('.wiki/'):
                clean_url = clean_url[6:]  # Remove '.wiki/' prefix
            return DOCS_DIR / clean_url

        # Handle relative paths
        source_dir = source_file.parent
        target = source_dir / clean_url

        try:
            return target.resolve()
        except:
            return target

    def file_exists(self, path: Path) -> bool:
        """Check if file exists, trying multiple extensions"""
        if path.exists():
            return True

        # Try with .md extension
        if path.with_suffix('.md').exists():
            return True

        # Try adding .md if no extension
        if not path.suffix:
            md_path = Path(str(path) + '.md')
            if md_path.exists():
                return True

        return False

    def analyze_file(self, filepath: Path):
        """Analyze a single markdown file"""
        self.total_files += 1

        try:
            content = filepath.read_text(encoding='utf-8')
        except:
            return

        links = self.extract_links(content)
        relative_path = filepath.relative_to(DOCS_DIR)

        for link_text, link_url in links:
            self.total_links += 1

            # External link
            if self.is_external(link_url):
                self.external_links.append({
                    'file': str(relative_path),
                    'text': link_text,
                    'url': link_url
                })
                continue

            # Anchor only
            if link_url.startswith('#'):
                self.anchor_only_links.append({
                    'file': str(relative_path),
                    'text': link_text,
                    'url': link_url
                })
                continue

            # Internal link - verify it exists
            target_path = self.resolve_path(link_url, filepath)

            if target_path is None:
                # Anchor only, already handled
                continue

            # Get target path string
            try:
                target_str = str(target_path.relative_to(DOCS_DIR)) if target_path else link_url
            except ValueError:
                # Path is outside wiki directory
                target_str = str(target_path) if target_path else link_url

            link_info = {
                'file': str(relative_path),
                'text': link_text,
                'url': link_url,
                'target': target_str
            }

            if self.file_exists(target_path):
                self.valid_links.append(link_info)
            else:
                # Try to find similar files
                suggestion = self.find_similar_file(target_path)
                link_info['suggestion'] = suggestion
                self.broken_links.append(link_info)

    def find_similar_file(self, target: Path) -> str:
        """Find similar filename in wiki docs"""
        target_name = target.name.lower()

        # Search for files with similar names
        for file in DOCS_DIR.rglob('*.md'):
            if file.name.lower() == target_name or \
               file.name.lower() == f"{target_name}.md" or \
               file.stem.lower() == target_name.replace('.md', '').lower():
                return str(file.relative_to(DOCS_DIR))

        # Search for partial matches
        target_stem = target.stem.lower()
        for file in DOCS_DIR.rglob('*.md'):
            if target_stem in file.stem.lower():
                return str(file.relative_to(DOCS_DIR))

        return "No similar file found"

    def analyze_all(self):
        """Analyze all markdown files"""
        print("Scanning documentation files...")

        md_files = sorted(DOCS_DIR.rglob('*.md'))
        for filepath in md_files:
            print(f"  Analyzing: {filepath.relative_to(DOCS_DIR)}")
            self.analyze_file(filepath)

        print(f"\nScan complete! Analyzed {self.total_files} files")

    def generate_report(self):
        """Generate comprehensive report"""
        with open(REPORT_FILE, 'w') as f:
            f.write("# Documentation Link Audit Report\n\n")
            f.write("Generated for wiki compatibility verification\n\n")

            # Statistics
            f.write("## Statistics\n\n")
            f.write(f"- **Total Files Scanned**: {self.total_files}\n")
            f.write(f"- **Total Links Found**: {self.total_links}\n")
            f.write(f"- **Valid Internal Links**: {len(self.valid_links)}\n")
            f.write(f"- **Broken Internal Links**: {len(self.broken_links)}\n")
            f.write(f"- **External Links**: {len(self.external_links)}\n")
            f.write(f"- **Anchor-Only Links**: {len(self.anchor_only_links)}\n")

            if self.total_links > 0:
                health = (len(self.valid_links) * 100) // self.total_links
                f.write(f"- **Health Score**: {health}%\n\n")

            # Broken links
            if self.broken_links:
                f.write("## Broken Links\n\n")
                f.write(f"Found {len(self.broken_links)} broken internal links:\n\n")
                f.write("| File | Link Text | Target URL | Suggestion |\n")
                f.write("|------|-----------|------------|------------|\n")

                for link in self.broken_links:
                    f.write(f"| {link['file']} | {link['text']} | `{link['url']}` | {link['suggestion']} |\n")

                f.write("\n")
            else:
                f.write("## Broken Links\n\n")
                f.write("**No broken links found!** All internal links are valid.\n\n")

            # Wiki compatibility recommendations
            f.write("## Wiki Compatibility Issues\n\n")

            # Find links that need updating for wiki format
            needs_update = []
            for link in self.valid_links:
                url = link['url']
                # Check for .md extension
                if url.endswith('.md'):
                    needs_update.append({
                        **link,
                        'issue': 'Contains .md extension',
                        'fix': url[:-3]  # Remove .md
                    })
                # Check for absolute paths
                elif url.startswith('/'):
                    needs_update.append({
                        **link,
                        'issue': 'Uses absolute path',
                        'fix': 'Convert to relative path'
                    })

            if needs_update:
                f.write(f"Found {len(needs_update)} links that should be updated for wiki compatibility:\n\n")
                f.write("| File | Current URL | Issue | Suggested Fix |\n")
                f.write("|------|-------------|-------|---------------|\n")

                for link in needs_update[:50]:  # Limit to 50 examples
                    f.write(f"| {link['file']} | `{link['url']}` | {link['issue']} | `{link['fix']}` |\n")

                if len(needs_update) > 50:
                    f.write(f"\n*... and {len(needs_update) - 50} more*\n")

                f.write("\n")

            # External links sample
            f.write("## External Links\n\n")
            f.write(f"Found {len(self.external_links)} external links. ")
            f.write("Sample of external links:\n\n")

            for link in self.external_links[:10]:
                f.write(f"- [{link['text']}]({link['url']}) in `{link['file']}`\n")

            if len(self.external_links) > 10:
                f.write(f"\n*... and {len(self.external_links) - 10} more*\n")

            # Recommendations
            f.write("\n## Recommendations\n\n")
            f.write("### For Wiki Compatibility\n\n")
            f.write("1. **Remove .md extensions**: Wiki links should be `[Text](Page)` not `[Text](Page.md)`\n")
            f.write("2. **Use relative paths**: Prefer `../folder/Page` over `/.wiki/folder/Page`\n")
            f.write("3. **Fix broken links**: Update or remove the broken links listed above\n")
            f.write("4. **Test in wiki**: Verify links work in GitHub Wiki environment\n\n")

            f.write("### Next Steps\n\n")
            if self.broken_links:
                f.write(f"1. Fix {len(self.broken_links)} broken links\n")
            if needs_update:
                f.write(f"2. Update {len(needs_update)} links to wiki format\n")
            f.write("3. Re-run this audit to verify fixes\n")
            f.write("4. Update `_Sidebar.md` with corrected paths\n")

        print(f"\nReport saved to: {REPORT_FILE}")

if __name__ == '__main__':
    analyzer = LinkAnalyzer()
    analyzer.analyze_all()
    analyzer.generate_report()

    # Print summary
    print("\n=== Summary ===")
    print(f"Files scanned:     {analyzer.total_files}")
    print(f"Total links:       {analyzer.total_links}")
    print(f"Valid links:       {len(analyzer.valid_links)}")
    print(f"Broken links:      {len(analyzer.broken_links)}")
    print(f"External links:    {len(analyzer.external_links)}")

    if analyzer.broken_links:
        print(f"\n✗ Found {len(analyzer.broken_links)} broken links")
        print(f"See report: {REPORT_FILE}")
    else:
        print("\n✓ All internal links are valid!")
