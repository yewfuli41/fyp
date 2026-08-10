import { Alert, Button, Modal } from "react-bootstrap";

interface ConfirmDeleteModalProps {
    show: boolean;
    title: string;
    itemName?: string;
    // Extra warning shown below the main prompt — e.g. to call out cascading
    // effects of the deletion that aren't obvious from the item's name alone.
    warningNote?: string;
    error: string;
    isDeleting: boolean;
    onCancel: () => void;
    onConfirm: () => void;
}

// Generic "are you sure you want to delete X?" confirmation dialog, shared by
// pages that each used to carry their own near-identical copy (Staff, Service, ...).
export default function ConfirmDeleteModal({
    show, title, itemName, warningNote, error, isDeleting, onCancel, onConfirm,
}: ConfirmDeleteModalProps) {
    return (
        <Modal show={show} onHide={onCancel}>
            <Modal.Header closeButton>
                <Modal.Title>{title}</Modal.Title>
            </Modal.Header>
            <Modal.Body>
                {error ? (
                    <Alert variant="danger" className="mb-0">{error}</Alert>
                ) : (
                    <>
                        <p className={warningNote ? "mb-2" : "mb-0"}>
                            Are you sure you want to delete <strong>{itemName}</strong>?
                            This action cannot be undone.
                        </p>
                        {warningNote && (
                            <p className="text-muted small mb-0">{warningNote}</p>
                        )}
                    </>
                )}
            </Modal.Body>
            <Modal.Footer>
                <Button variant="outline-secondary" onClick={onCancel}>
                    {error ? "Close" : "Cancel"}
                </Button>
                {!error && (
                    <Button variant="danger" onClick={onConfirm} disabled={isDeleting}>
                        {isDeleting ? "Deleting..." : "Delete"}
                    </Button>
                )}
            </Modal.Footer>
        </Modal>
    );
}
