#!/usr/bin/env python3
"""Fail closed until the redistribution evidence for the actual FFmpeg bundle exists."""
import hashlib
import json
from pathlib import Path

root = Path(__file__).resolve().parent.parent
manifest = json.loads((root / 'docs/ffmpeg-manifest.json').read_text())
if manifest.get('redistribution_verified') is not True:
    raise SystemExit('Release bloqueada: faltam fontes correspondentes e notices verificados do build FFmpeg completo. A pasta build/tools é apenas para teste local.')
for name, digest in manifest['files'].items():
    path = root / 'build/tools' / name
    if not path.is_file() or hashlib.sha256(path.read_bytes()).hexdigest() != digest:
        raise SystemExit(f'Ferramenta divergente ou ausente: {name}')
for field in ('source_evidence', 'dependency_notices', 'configuration_evidence'):
    value = manifest.get(field)
    if not value or not (root / value).is_file():
        raise SystemExit(f'Evidência ausente: {field}')
if not (root / 'build/DepoimentoStudio.exe').is_file():
    raise SystemExit('Compile o executável antes de empacotar.')
print('Evidências declaradas e hashes presentes. Ainda requer aprovação funcional Windows e revisão das licenças.')
