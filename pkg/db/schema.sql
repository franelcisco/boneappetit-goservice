-- public.suscription_logs definition
-- Drop table
-- DROP TABLE public.suscription_logs;
CREATE TABLE public.suscription_logs
(
    id int4 GENERATED ALWAYS AS IDENTITY( INCREMENT BY 1 MINVALUE 1 MAXVALUE 2147483647 START 1 CACHE 1 NO CYCLE) NOT NULL,
    type character varying(128) NOT NULL,
    message character varying(512) NOT NULL,
    new_value character varying(256),
    old_value character varying(256),
    updated_field character varying(128),
    created_at TIMESTAMP WITHOUT TIME ZONE DEFAULT NOW(),
    order_id text NOT NULL,
    CONSTRAINT suscription_logs_pkey PRIMARY KEY (id)
);

ALTER TABLE public.suscription_logs ADD CONSTRAINT suscription_logs_orders_id_fkey FOREIGN KEY (order_id) REFERENCES public.orders (id) MATCH SIMPLE ON UPDATE CASCADE ON DELETE CASCADE;