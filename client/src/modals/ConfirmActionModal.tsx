import type { ReactNode } from "react";
import { Alert, Button, Modal } from "react-bootstrap";

interface ConfirmActionModalProps {
    show: boolean;
    title: string;
    body: ReactNode;
    confirmLabel: string;
    confirmingLabel: string;
    confirmVariant?: string;
    error?: string;
    isBusy: boolean;
    onCancel: () => void;
    onConfirm: () => void;
}

// Generic "are you sure?" confirmation for a business action on an existing
// booking (reject a request, reject a reschedule proposal, cancel a booking)
// — distinct from ConfirmDeleteModal, which is worded specifically around
// deleting an item rather than acting on a booking.
export default function ConfirmActionModal({
    show, title, body, confirmLabel, confirmingLabel, confirmVariant = "danger",
    error, isBusy, onCancel, onConfirm,
}: ConfirmActionModalProps) {
    return (
        <Modal show={show} onHide={onCancel} backdrop="static">
            <Modal.Header closeButton><Modal.Title>{title}</Modal.Title></Modal.Header>
            <Modal.Body>
                {body}
                {error && <Alert variant="danger" className="py-2 mb-0 mt-3">{error}</Alert>}
            </Modal.Body>
            <Modal.Footer>
                <Button variant="outline-secondary" onClick={onCancel} disabled={isBusy}>Keep it</Button>
                <Button variant={confirmVariant} onClick={onConfirm} disabled={isBusy}>
                    {isBusy ? confirmingLabel : confirmLabel}
                </Button>
            </Modal.Footer>
        </Modal>
    );
}
