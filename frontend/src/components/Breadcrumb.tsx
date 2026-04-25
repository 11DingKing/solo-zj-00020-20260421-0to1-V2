import { BreadcrumbItem } from "../types";

interface BreadcrumbProps {
  items: BreadcrumbItem[];
  onNavigate: (folderId: number) => void;
}

const Breadcrumb = ({ items, onNavigate }: BreadcrumbProps) => {
  return (
    <div className="breadcrumb">
      {items.map((item, index) => {
        const isLast = index === items.length - 1;

        return (
          <span key={item.id}>
            {index > 0 && <span className="breadcrumb-separator"> / </span>}
            {isLast ? (
              <span className="breadcrumb-item current">{item.name}</span>
            ) : (
              <span
                className="breadcrumb-item"
                onClick={() => onNavigate(item.id)}
              >
                {item.name}
              </span>
            )}
          </span>
        );
      })}
    </div>
  );
};

export default Breadcrumb;
