#!/bin/bash
# Script to set up wiki as a submodule (run once)

# Clone wiki separately first
git clone https://github.com/acamarata/nself.wiki.git ../nself-wiki

# Add as submodule
git submodule add https://github.com/acamarata/nself.wiki.git .wiki

# Create sync script
cat > sync-wiki.sh << 'EOF'
#!/bin/bash
# Sync /docs to wiki submodule

# Copy docs
cp -r docs/* .wiki/

# Convert filenames
cd .wiki
for file in *.MD *.md; do
  if [[ -f "$file" ]]; then
    newname=$(echo "$file" | sed 's/_/-/g')
    [[ "$file" != "$newname" ]] && mv "$file" "$newname"
  fi
done

# Commit and push
git add .
git commit -m "Sync from docs"
git push

cd ..
git add .wiki
git commit -m "Update wiki submodule"
EOF

chmod +x sync-wiki.sh
echo "Run ./sync-wiki.sh to sync docs to wiki"