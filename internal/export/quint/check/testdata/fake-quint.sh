#!/bin/sh
# A fake quint for the engquint tests. It prints canned output shaped like
# Quint 0.33.1 and fails loudly when a mandatory flag is missing, so the tests
# prove that engquint passes them. Behavior is selected by environment:
#   FAKE_QUINT_LOG          append one line per invocation (args and environment)
#   FAKE_QUINT_VERSION      output of --version (default 0.33.1)
#   FAKE_QUINT_VIOLATE      invariant val name that is violated (run and verify)
#   FAKE_QUINT_RUNTIME_ERR  invariant val name whose run fails with QNT502
#   FAKE_QUINT_TRACE_MAX    "max=" of the trace length statistics (default 7)
#   FAKE_QUINT_WIT_ZERO     witness val name witnessed in 0 traces
#   FAKE_QUINT_TYPE_ERROR   non-empty: typecheck fails with a located error
#   FAKE_QUINT_SEQUENCE     the counterexample (default "TR-A TR-B TR-C")
cmd="$1"
shift
if [ "$cmd" = "--version" ]; then
  echo "${FAKE_QUINT_VERSION:-0.33.1}"
  exit 0
fi

backend= main= seed= samples= steps= inv= itf= out= endpoint= config= wits= file=
in_wits=
for a in "$@"; do
  case "$a" in
    --witnesses) in_wits=1 ;;
    --backend=*) backend="${a#--backend=}"; in_wits= ;;
    --main=*) main="${a#--main=}"; in_wits= ;;
    --seed=*) seed="${a#--seed=}"; in_wits= ;;
    --max-samples=*) samples="${a#--max-samples=}"; in_wits= ;;
    --max-steps=*) steps="${a#--max-steps=}"; in_wits= ;;
    --invariant=*) inv="${a#--invariant=}"; in_wits= ;;
    --out-itf=*) itf="${a#--out-itf=}"; in_wits= ;;
    --out=*) out="${a#--out=}"; in_wits= ;;
    --server-endpoint=*) endpoint="${a#--server-endpoint=}"; in_wits= ;;
    --apalache-config=*) config="${a#--apalache-config=}"; in_wits= ;;
    --apalache-version*) echo "FAKE QUINT: --apalache-version must not be passed" >&2; exit 99 ;;
    --*) in_wits= ;;
    *) if [ -n "$in_wits" ]; then wits="$wits $a"; else file="$a"; fi ;;
  esac
done

if [ -n "$FAKE_QUINT_LOG" ]; then
  echo "$cmd file=$file backend=$backend main=$main seed=$seed samples=$samples steps=$steps inv=$inv wits=[${wits# }] endpoint=$endpoint config=$config QUINT_HOME=$QUINT_HOME NO_PROXY=$NO_PROXY no_proxy=$no_proxy TMPDIR=$TMPDIR PWD=$PWD" >> "$FAKE_QUINT_LOG"
fi

write_itf() {
  [ -n "$itf" ] || return 0
  printf '{"#meta":{"format":"ITF","status":"violation"},"vars":["lastTransition"],"states":[{"#meta":{"index":0},"lastTransition":"init"}' > "$itf"
  i=1
  for tr in ${FAKE_QUINT_SEQUENCE:-TR-A TR-B TR-C}; do
    printf ',{"#meta":{"index":%s},"lastTransition":"%s"}' "$i" "$tr" >> "$itf"
    i=$((i + 1))
  done
  printf ']}\n' >> "$itf"
}

case "$cmd" in
  typecheck)
    if [ -n "$FAKE_QUINT_TYPE_ERROR" ]; then
      [ -n "$out" ] && printf '{"stage":"typechecking","errors":[{"explanation":"[QNT000] Couldn'"'"'t unify int and str\\nTrying to unify int and str","locs":[{"source":"%s","start":{"line":2,"col":4,"index":10},"end":{"line":2,"col":9,"index":15}}]}]}\n' "$file" > "$out"
      echo "error: type error" >&2
      exit 1
    fi
    [ -n "$out" ] && echo '{"stage":"typechecking","errors":[]}' > "$out"
    exit 0
    ;;
  run)
    if [ "$backend" != "typescript" ]; then
      echo "FAKE QUINT: run without --backend=typescript (got '$backend')" >&2
      exit 99
    fi
    for pair in "main=$main" "seed=$seed" "max-samples=$samples" "max-steps=$steps"; do
      case "$pair" in
        *=) echo "FAKE QUINT: run without --${pair%=}" >&2; exit 99 ;;
      esac
    done
    if [ -n "$inv" ] && [ "$inv" = "$FAKE_QUINT_RUNTIME_ERR" ]; then
      echo "[error] Runtime error QNT502: variable v_attempts is not set (12ms)" >&2
      exit 1
    fi
    if [ -n "$inv" ] && [ "$inv" = "$FAKE_QUINT_VIOLATE" ]; then
      write_itf
      echo "[violation] Found an issue (8ms at 125 traces/second)."
      echo "error: Invariant violated" >&2
      exit 1
    fi
    echo "[ok] No violation found (260ms at 3846 traces/second)."
    echo "Trace length statistics: max=${FAKE_QUINT_TRACE_MAX:-7}, min=1, average=4.43"
    if [ -n "$wits" ]; then
      echo "Witnesses:"
      for w in $wits; do
        if [ "$w" = "$FAKE_QUINT_WIT_ZERO" ]; then
          echo "$w was witnessed in 0 trace(s) out of $samples explored (0.00%)"
        else
          echo "$w was witnessed in 17 trace(s) out of $samples explored (1.70%)"
        fi
      done
    fi
    exit 0
    ;;
  verify)
    case "$endpoint" in
      localhost:[0-9]*) ;;
      *) echo "FAKE QUINT: verify without --server-endpoint=localhost:<port>" >&2; exit 99 ;;
    esac
    if [ -z "$QUINT_HOME" ] || [ -z "$config" ] || [ ! -f "$config" ]; then
      echo "FAKE QUINT: verify without QUINT_HOME or a readable --apalache-config" >&2
      exit 99
    fi
    if [ "$TMPDIR" != "$PWD" ]; then
      echo "FAKE QUINT: verify must run with TMPDIR equal to the working directory" >&2
      exit 99
    fi
    if [ -n "$inv" ] && [ "$inv" = "$FAKE_QUINT_VIOLATE" ]; then
      write_itf
      echo "[violation] Found an issue (5203ms)."
      exit 1
    fi
    echo "[ok] No violation found (4100ms)."
    exit 0
    ;;
esac
echo "FAKE QUINT: unsupported command '$cmd'" >&2
exit 99
