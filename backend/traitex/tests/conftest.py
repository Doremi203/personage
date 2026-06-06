import os
import sys

# Make the traitex root importable so test files can use the same `app.…` /
# `proto.…` paths the service uses at runtime.
ROOT = os.path.abspath(os.path.join(os.path.dirname(__file__), os.pardir))
if ROOT not in sys.path:
    sys.path.insert(0, ROOT)

# Generated gRPC stubs import their siblings by bare module name
# (e.g. `import state_tracking_pb2`), so their directories must be on the path.
# Mirror the runtime PYTHONPATH set in the Dockerfile.
for _proto_dir in ("proto", os.path.join("externalClients", "personage_auth", "proto")):
    _abs = os.path.join(ROOT, _proto_dir)
    if os.path.isdir(_abs) and _abs not in sys.path:
        sys.path.insert(0, _abs)
