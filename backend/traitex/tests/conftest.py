import os
import sys

# Make the traitex root importable so test files can use the same `app.…` /
# `proto.…` paths the service uses at runtime.
ROOT = os.path.abspath(os.path.join(os.path.dirname(__file__), os.pardir))
if ROOT not in sys.path:
    sys.path.insert(0, ROOT)
