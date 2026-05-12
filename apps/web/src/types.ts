export type ComprobanteRow = {
  id: string;
  tipo: string;
  serie: string;
  correlativo: number;
  fecha_emision: string;
  moneda: string;
  receptor_doc: string;
  receptor_razon: string;
  total: number;
  igv: number;
  estado: string;
  sunat_codigo?: string;
  sunat_mensaje?: string;
  hash_cpe?: string;
  created_at: string;
};

export type ComprobanteDetalleT = ComprobanteRow & {
  gravado: number;
  exonerado: number;
  inafecto: number;
  payload: any;
};

export type Item = {
  codigo?: string;
  descripcion: string;
  unidad: string;
  cantidad: number;
  valor_unitario: number;
  afectacion_igv: string;
};

export type Cliente = {
  id?: string;
  tipo_doc: string;
  num_doc: string;
  razon_social: string;
  direccion?: string;
  email?: string;
  telefono?: string;
  created_at?: string;
};

export type Producto = {
  id?: string;
  codigo?: string;
  descripcion: string;
  unidad: string;
  valor_unitario: number;
  afectacion_igv: string;
  created_at?: string;
};

export type Totales = {
  gravado: number;
  exonerado: number;
  inafecto: number;
  gratuito: number;
  igv: number;
  isc: number;
  icbper: number;
  total: number;
};
