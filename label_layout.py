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


def encode_qr_payload(data: dict, doc_type: str, sheet_idx: int, total: int) -> str:
    """Build the pipe-delimited QR payload string for a label sheet.

    Args:
        data: parsed label request data (from document_schema.parse_label_request)
        doc_type: '1' for equipment, '2' for project
        sheet_idx: 1-based sheet number (e.g., 1 for first sheet)
        total: total sheet count

    Returns:
        Pipe-delimited string matching the historical auto-generate format.
    """
    count_str = f"{sheet_idx}/{total}"
    if doc_type == '1':
        return "|".join([
            str(data['eq_number']),
            str(data['eq_doc_number']),
            str(data['eq_doc_title']),
            str(data['eq_doc_department']),
            str(data['eq_doc_year']),
            count_str,
        ])
    else:
        return "|".join([
            str(data['pjt_number']),
            str(data['pjt_test_number']),
            str(data['pjt_doc_title']),
            str(data['pjt_doc_writer']),
            count_str,
        ])
