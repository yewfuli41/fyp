import { Alert, Button, Modal } from "react-bootstrap";

interface ConfirmDeleteModalProps {
    show: boolean;
    title: string;
    itemName?: string;
    error: string;
    isDeleting: boolean;
    onCancel: () => void;
    onConfirm: () => void;
}

// Generic "are you sure you want to delete X?" confirmation dialog, shared by
// pages that each used to carry their own near-identical copy (Staff, Service, ...).
export default function ConfirmDeleteModal({
    show, title, itemName, error, isDeleting, onCancel, onConfirm,
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
                    <p className="mb-0">
                        Are you sure you want to delete <strong>{itemName}</strong>?
                        This action cannot be undone.
                    </p>
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
