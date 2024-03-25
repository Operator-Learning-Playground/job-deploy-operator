set -o errexit
set -o nounset
set -o pipefail

HELM_ROOT=$(dirname "${BASH_SOURCE}")/..
cd "${HELM_ROOT}"

find_files() {
  find . -not \( \
      \( \
        -wholename './output' \
        -o -wholename '*/vendor/*' \
      \) -prune \
    \) -name '*.go'
}

GOFMT="gofmt -s"

bad_files=$(find_files | xargs $GOFMT -l)
if [[ -n "${bad_files}" ]]; then
  echo "Please run hack/update-gofmt.sh to fix the following files:"
  echo "${bad_files}"
  exit 1
fi