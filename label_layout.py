"""
Label layout configuration: binder size → QR cell position and column width.
"""

_BINDER_QR_CONFIG = {
    7: {'column_width': 1.875, 'equipment_qr_cell': 'E9', 'project_qr_cell': 'E8'},
    5: {'column_width': 1.25,  'equipment_qr_cell': 'D9', 'project_qr_cell': 'D8'},
    3: {'column_width': 1,     'equipment_qr_cell': 'D9', 'project_qr_cell': 'D8'},
    1: {'column_width': 0.75,  'equipment_qr_cell': 'B9', 'project_qr_cell': 'B9'},
}

_DEFAULT_BINDER_SIZE = 3


def get_qr_config(doc_type: str, binder_size: int) -> dict:
    """Return QR placement config for the given doc_type and binder_size.

    Returns a dict with keys:
      - 'column_width': float — width for columns B–M
      - 'cell_pos': str — Excel cell address for QR image anchor

    Falls back to binder_size=3 if the size is not found.
    """
    entry = _BINDER_QR_CONFIG.get(binder_size, _BINDER_QR_CONFIG[_DEFAULT_BINDER_SIZE])
    cell_key = 'equipment_qr_cell' if doc_type == '1' else 'project_qr_cell'
    return {
        'column_width': entry['column_width'],
        'cell_pos': entry[cell_key],
    }
