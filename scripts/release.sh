version=`jq -r '.version' package.json`
outfile="releases/dadbom-$1-$version.zip"
if test -f "$outfile"; then
    echo "This release version already exists locally: $outfile"
    exit 1
fi
cd dist
zip -r "../$outfile" *
cd ..